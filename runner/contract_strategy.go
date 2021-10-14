package runner

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"crypto-trading-bot-engine/message"
	"crypto-trading-bot-engine/strategy"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
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

	// Status control
	CheckPriceEnabled bool // Don't check price when runner is gonna to stop
	stopChClosed      bool // Let handler know whether StopCh has been closed

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

	// Make sure there is only one task processed at one time
	RunnerMutex sync.Mutex

	// Send the notification, only support telegram atm
	sender message.Messenger // all users use the same one, but sent with different chat_id

	// Check mark price once a time
	ignoreIncomingMark bool

	// Last time the price being checked
	LastPriceCheckedTime time.Time
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
		ContractStrategy:  cs,
		contract:          c,
		contractHook:      ch,
		StopCh:            make(chan bool),
		MarkCh:            make(chan contract.Mark),
		CheckPriceEnabled: true,
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

func (r *ContractStrategyRunner) Stop() {
	if !r.stopChClosed {
		// In order to avoid `panic: close of closed channel`, check first
		r.stopChClosed = true

		// This is necessary, if stopCh sent first and broadcastMark keeps sending Mark to nowhere, it would get stuck
		// If runner MarhCh get stuck, handler stopContractStrategyRunner will get stuck too, it would cause that
		// runner beforeCloseFunc can't be finished and gets stuck. Basically, it's a deadlock hell
		r.CheckPriceEnabled = false
		close(r.StopCh)
	}
}

// Start
func (r *ContractStrategyRunner) Run() {
	// NOTE For graceful shutdown
	r.handlerBlockWg.Add(1)
	defer r.handlerBlockWg.Done()

	halted := false
	for {
		select {
		case <-r.StopCh:
			halted = true
			break
		case mark := <-r.MarkCh:
			// If 'CheckPrice' is still in progress, ignore the incoming prices unitl it's finished
			if !r.CheckPriceEnabled || r.ignoreIncomingMark {
				break
			}
			r.ignoreIncomingMark = true
			go r.checkPrice(&mark)
		}
		if halted {
			break
		}
	}

	r.RunnerBlockWg.Wait()
	r.beforeCloseFunc(r.ContractStrategy.Symbol, r.ContractStrategy.Uuid)
}

// Check mark price
func (r *ContractStrategyRunner) checkPrice(mark *contract.Mark) {
	r.RunnerMutex.Lock()
	defer r.RunnerMutex.Unlock()

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
	// r.logger.Println(r.ContractStrategy.Symbol, r.ContractStrategy.Uuid, mark.Time.Format("2006-01-02 15:04:05"), mark.Price)

	// Make sure the data is valid
	if err := r.validateExchangeOrdersDetails(); err != nil {
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' invalid 'exchange_orders_details', err: %s\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus), err)
		r.CheckPriceEnabled = false
		r.handlerEventsCh.OutOfSync <- r.ContractStrategy.Uuid
		r.handlerEventsCh.Disable <- r.ContractStrategy.Uuid
		return
	}

	halted, err := r.contract.CheckPrice(*mark)
	if err != nil && halted { // scenario: DB fails
		// Stop receiving Mark
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' halted with err: %s\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus), err)
		r.CheckPriceEnabled = false
		r.handlerEventsCh.OutOfSync <- r.ContractStrategy.Uuid
		r.handlerEventsCh.Disable <- r.ContractStrategy.Uuid
	} else if err != nil { // scenario: ftx api 400, still want to retry
		r.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' err: %v\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus), err)

		// Sleep a while and try again
		time.Sleep(time.Second * 3)
	} else if halted { // scenario: take-profit, err is nil
		// Stop receiving Mark
		r.CheckPriceEnabled = false
		r.logger.Printf("[INFO] strategy: '%s', user: '%s', symbol: '%s', positionStatus: '%s' is done!\n", r.ContractStrategy.Uuid, r.ContractStrategy.UserUuid, r.ContractStrategy.Symbol, contract.TranslateStatusByInt(r.ContractStrategy.PositionStatus))
		r.handlerEventsCh.Reset <- r.ContractStrategy.Uuid
	}

	r.LastPriceCheckedTime = time.Now()
}

// Check exchange_orders_details, halt the strategy if the data is out of sync
func (r *ContractStrategyRunner) validateExchangeOrdersDetails() error {
	switch contract.Status(r.ContractStrategy.PositionStatus) {
	case contract.CLOSED:
		if len(r.ContractStrategy.ExchangeOrdersDetails) > 0 {
			return errors.New("position status: 'CLOSED', 'exchange_orders_details' isn't empty")
		}
	case contract.OPENED:
		if len(r.ContractStrategy.ExchangeOrdersDetails) == 0 {
			return errors.New("position status: 'OPENED', 'exchange_orders_details' is empty")
		}
		entryOrder, ok := r.ContractStrategy.ExchangeOrdersDetails["entry_order"].(map[string]interface{})
		if !ok {
			return errors.New("position status: 'OPENED', 'exchange_orders_details.entry_order' is missing")
		}
		_, ok = entryOrder["order_id"]
		if !ok {
			return errors.New("position status: 'OPENED', 'exchange_orders_details.entry_order.order_id' is missing")
		}
	case contract.UNKNOWN:
		return errors.New("unknown status")
	default:
		return errors.New("undefined status")
	}
	return nil
}
