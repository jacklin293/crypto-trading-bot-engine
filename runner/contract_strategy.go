package runner

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/exchange"
	"crypto-trading-bot-main/message"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"
)

type ContractStrategyRunner struct {
	contractStrategy *db.ContractStrategy
	user             *db.User

	logger *log.Logger

	// Channel
	StopCh chan bool
	MarkCh chan contract.Mark

	// DB
	db *db.DB

	// Support contract only atm
	contract     *contract.Contract
	contractHook *contractHook

	// Disable strategy
	disableCh chan db.ContractStrategy

	// Process the strategy out of sync
	outOfSyncCh chan db.ContractStrategy

	// reset strategy
	resetCh chan db.ContractStrategy

	// Deal with the sig for stopping the strategy
	runnerBlockWg   *sync.WaitGroup
	beforeCloseFunc func(string, string)

	// Send the notification, only support telegram atm
	sender message.Messenger // all users use the same one, but sent with different chat_id

	// Check mark price once a time
	ignoreIncomingMark bool
}

func NewContractStrategyRunner(cs *db.ContractStrategy) (*ContractStrategyRunner, error) {
	// New contract hook
	ch := newContractHook(cs)
	ch.contractStrategy = cs

	// New contract
	c, err := contract.NewContract(order.Side(cs.Side), cs.Params)
	if err != nil {
		return &ContractStrategyRunner{}, err
	}
	c.SetHook(ch)
	c.SetStatus(contract.Status(cs.PositionStatus))

	s := &ContractStrategyRunner{
		contractStrategy: cs,
		contract:         c,
		contractHook:     ch,
		StopCh:           make(chan bool),
		MarkCh:           make(chan contract.Mark),
	}
	return s, err
}

func (r *ContractStrategyRunner) SetLogger(l *log.Logger) {
	r.logger = l
	r.contractHook.setLogger(r.logger)
}

func (r *ContractStrategyRunner) SetDB(db *db.DB) {
	r.db = db
	r.contractHook.db = db
}

func (r *ContractStrategyRunner) SetBeforeCloseFunc(f func(string, string)) {
	r.beforeCloseFunc = f
}

func (r *ContractStrategyRunner) SetRunnerBlockWg(wg *sync.WaitGroup) {
	r.runnerBlockWg = wg
}

func (r *ContractStrategyRunner) SetSymbolEntryTakenMutexForHook(m map[string]*sync.Mutex) {
	r.contractHook.setSymbolEntryTakenMutex(m)
}

func (r *ContractStrategyRunner) SetExchangeForHook(ex exchange.Exchanger) {
	r.contractHook.setExchange(ex)
}

func (r *ContractStrategyRunner) SetSender(m message.Messenger) {
	r.sender = m
	r.contractHook.setSender(m)
}

func (r *ContractStrategyRunner) SetUser(u *db.User) {
	r.user = u
	r.contractHook.setUser(u)
}

func (r *ContractStrategyRunner) SetDisableCh(ch chan db.ContractStrategy) {
	r.disableCh = ch
}

func (r *ContractStrategyRunner) SetOutOfSyncCh(ch chan db.ContractStrategy) {
	r.outOfSyncCh = ch
}

func (r *ContractStrategyRunner) SetResetCh(ch chan db.ContractStrategy) {
	r.resetCh = ch
}

// Start
func (r *ContractStrategyRunner) Run() {
	halted := false
	for {
		select {
		case <-r.StopCh:
			halted = true
			break
		case mark := <-r.MarkCh:
			// If 'CheckPrice' is still in progress, ignore the incoming prices unitl it's finished
			if r.ignoreIncomingMark {
				break
			}
			r.ignoreIncomingMark = true
			go r.checkPrice(&mark)
		}
		if halted {
			break
		}
	}

	// This might not be necessary, but better to wait until removal process done
	r.runnerBlockWg.Add(1)
	defer r.runnerBlockWg.Done()
	r.beforeCloseFunc(r.contractStrategy.Symbol, r.contractStrategy.Uuid)
}

// Check mark price
func (r *ContractStrategyRunner) checkPrice(mark *contract.Mark) {
	// for graceful shutdown, block everything in progress until they are done
	r.runnerBlockWg.Add(1)
	defer r.runnerBlockWg.Done()
	defer func() { r.ignoreIncomingMark = false }()
	defer func() {
		if e := recover(); e != nil {
			r.logger.Printf("strategy '%s' panic: %v stack: %s\n", r.contractStrategy.Uuid, e, string(debug.Stack()))
			// TODO telegram
			// TODO call r.disableStrategy
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(r.contractStrategy.Side), r.contractStrategy.Symbol)
			r.sender.Send(r.user.TelegramChatId, text)
			r.outOfSyncCh <- *r.contractStrategy
			r.disableCh <- *r.contractStrategy
		}
	}()

	// TODO DEBUG del
	// r.logger.Println(r.contractStrategy.Symbol, r.contractStrategy.Uuid, mark.Time.Format("2006-01-02 15:04:05"), mark.Price, runtime.NumGoroutine())

	// TODO If it's internal error, sleep random secs
	halted, err := r.contract.CheckPrice(*mark)
	if err != nil && halted { // scenario: DB fails
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' - halted, err: %s\n", r.contractStrategy.Uuid, r.contractStrategy.UserUuid, r.contractStrategy.Symbol, contract.TranslateStatusByInt(r.contractStrategy.PositionStatus), err)
		r.outOfSyncCh <- *r.contractStrategy
		r.disableCh <- *r.contractStrategy

		// TODO FIXME panic: close of closed channel stack
		// TODO reproduction: create a scenario that sleep during the runtime then trigger `ctrl+c` to `close(ch)` in handler. The panic will be trigger by closing closed ch again
		close(r.StopCh)
	} else if err != nil { // scenario: ftx api 400, still want to retry
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' - err: %v\n", r.contractStrategy.Uuid, r.contractStrategy.UserUuid, r.contractStrategy.Symbol, contract.TranslateStatusByInt(r.contractStrategy.PositionStatus), err)

		// Sleep a while and try again
		time.Sleep(time.Second * 3)
	} else if halted { // scenario: take-profit, err is nil
		r.logger.Printf("[INFO] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' halted\n", r.contractStrategy.Uuid, r.contractStrategy.UserUuid, r.contractStrategy.Symbol, contract.TranslateStatusByInt(r.contractStrategy.PositionStatus))
		r.resetCh <- *r.contractStrategy
		close(r.StopCh)
	}

	// TODO telegram, Send some message after a period of time to user for indicating it's still alive

}
