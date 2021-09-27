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
	"crypto-trading-bot-main/strategy"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"
)

const (
	ALIVE_NOTIFICATION_INTERVAL = 60
)

type ContractStrategyRunner struct {
	ContractStrategy *db.ContractStrategy
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

	// Deal with the sig for stopping the strategy, all strategies share the same sync.WaitGroup
	handlerBlockWg  *sync.WaitGroup
	beforeCloseFunc func(string, string)

	// To disable/reset, etc.. a strategy from handler
	handlerEventsCh *strategy.EventsCh

	// Make sure strategy finishes its work before being killed
	RunnerBlockWg sync.WaitGroup

	// Send the notification, only support telegram atm
	sender message.Messenger // all users use the same one, but sent with different chat_id

	// Check mark price once a time
	ignoreIncomingMark bool

	// Alive notification
	lastAliveNotificationTime time.Time
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
		ContractStrategy:          cs,
		contract:                  c,
		contractHook:              ch,
		StopCh:                    make(chan bool),
		MarkCh:                    make(chan contract.Mark),
		lastAliveNotificationTime: time.Now(), // Don't send when it just launches
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

func (r *ContractStrategyRunner) SetHandlerBlockWg(wg *sync.WaitGroup) {
	r.handlerBlockWg = wg
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

func (r *ContractStrategyRunner) SetHandlerEventsCh(ch *strategy.EventsCh) {
	r.handlerEventsCh = ch
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
	r.handlerBlockWg.Add(1)
	defer r.handlerBlockWg.Done()
	r.RunnerBlockWg.Wait()
	r.beforeCloseFunc(r.ContractStrategy.Symbol, r.ContractStrategy.Uuid)
}

// Check mark price
func (r *ContractStrategyRunner) checkPrice(mark *contract.Mark) {
	// for graceful shutdown, block everything in progress until they are done
	r.handlerBlockWg.Add(1)
	defer r.handlerBlockWg.Done()
	defer func() { r.ignoreIncomingMark = false }()
	defer func() {
		if e := recover(); e != nil {
			r.logger.Printf("strategy '%s' panic: %v stack: %s\n", r.ContractStrategy.Uuid, e, string(debug.Stack()))
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(r.ContractStrategy.Side), r.ContractStrategy.Symbol)
			go r.sender.Send(r.user.TelegramChatId, text)
			r.handlerEventsCh.OutOfSync <- r.ContractStrategy.Uuid
			r.handlerEventsCh.Disable <- r.ContractStrategy.Uuid
		}
	}()

	// NOTE For DEBUG
	// r.logger.Println(r.ContractStrategy.Symbol, r.ContractStrategy.Uuid, mark.Time.Format("2006-01-02 15:04:05"), mark.Price, runtime.NumGoroutine())

	halted, err := r.contract.CheckPrice(*mark)
	if err != nil && halted { // scenario: DB fails
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' - halted, err: %s\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus), err)
		r.handlerEventsCh.OutOfSync <- r.ContractStrategy.Uuid
		r.handlerEventsCh.Disable <- r.ContractStrategy.Uuid
	} else if err != nil { // scenario: ftx api 400, still want to retry
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' - err: %v\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus), err)

		// Sleep a while and try again
		time.Sleep(time.Second * 3)
	} else if halted { // scenario: take-profit, err is nil
		r.logger.Printf("[INFO] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' is done!\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus))
		r.handlerEventsCh.Reset <- r.ContractStrategy.Uuid
	}

	// Send 'alive' message after a period of time
	if time.Now().After(r.lastAliveNotificationTime.Add(time.Minute * time.Duration(ALIVE_NOTIFICATION_INTERVAL))) {
		r.lastAliveNotificationTime = time.Now()
		text := fmt.Sprintf("[Info] Don't worry! '%s %s $%s' is still alive", order.TranslateSideByInt(r.ContractStrategy.Side), r.ContractStrategy.Symbol, r.ContractStrategy.Margin)
		go r.sender.Send(r.user.TelegramChatId, text)
	}
}
