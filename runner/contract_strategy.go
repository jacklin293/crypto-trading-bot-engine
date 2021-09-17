package runner

import (
	"log"
	"sync"
	"time"

	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"

	"github.com/shopspring/decimal"
)

type Mark struct {
	Price decimal.Decimal
	Time  time.Time
}

type ContractStrategyRunner struct {
	contractStrategy db.ContractStrategy

	logger *log.Logger

	// Channel
	StopCh chan bool
	MarkCh chan Mark

	// DB
	db *db.DB

	// Support contract only atm
	contract     *contract.Contract
	contractHook *contractHook

	// Deal with stop the strategy
	handlerBlockWg  *sync.WaitGroup
	beforeCloseFunc func(string, string)

	// Check mark price once a time
	ignoreIncomingMark bool
}

// TODO Refactor here due to too many params
// TODO Receive DB strategy
func NewContractStrategyRunner(cs db.ContractStrategy) (*ContractStrategyRunner, error) {
	ch := newContractHook()
	c, err := contract.NewContract(order.ContractDirection(cs.ContractDirection), cs.ContractParams)
	if err != nil {
		return &ContractStrategyRunner{}, err
	}
	c.SetHook(ch)

	s := &ContractStrategyRunner{
		contractStrategy: cs,
		contract:         c,
		contractHook:     ch,
		StopCh:           make(chan bool),
		MarkCh:           make(chan Mark),
	}

	return s, err
}

func (r *ContractStrategyRunner) SetLogger(l *log.Logger) {
	r.logger = l
	r.contractHook.setLogger(r.logger)
}

func (r *ContractStrategyRunner) SetDB(db *db.DB) {
	r.db = db
}

func (r *ContractStrategyRunner) SetBeforeCloseFunc(f func(string, string)) {
	r.beforeCloseFunc = f
}

func (r *ContractStrategyRunner) SetHandlerBlockWg(wg *sync.WaitGroup) {
	r.handlerBlockWg = wg
}

func (r *ContractStrategyRunner) SetPositionStatus(status contract.Status) {
	r.contract.SetStatus(status)
	// TODO activePosition after SetStatus
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
	r.beforeCloseFunc(r.contractStrategy.Symbol, r.contractStrategy.Uuid)
}

// Check mark price
func (r *ContractStrategyRunner) checkPrice(mark *Mark) {
	defer func() {
		if e := recover(); e != nil {
			r.logger.Printf("strategy '%s' panic: %v\n", r.contractStrategy.Uuid, e)
			// TODO telegram
			// TODO call r.disableStrategy
		}
	}()

	// Prevent the process from being closed when anything is in progress
	r.handlerBlockWg.Add(1)
	defer r.handlerBlockWg.Done()

	// TODO del
	r.logger.Println(r.contractStrategy.Uuid, mark.Time.Format("2006-01-02 15:04:05"), mark.Price)

	err, halted := r.contract.CheckPrice(mark.Time, mark.Price)
	if err != nil {
		r.logger.Printf("[ERROR] strategy '%s' CheckPrice err: %v\n", r.contractStrategy.Uuid, err)
	}
	if halted {
		r.logger.Printf("[INFO] strategy '%s' CheckPrice halted\n", r.contractStrategy.Uuid)
		// TODO telegram
		// TODO call r.disableStrategy()
	}
	r.ignoreIncomingMark = false
}

// TODO
func (r *ContractStrategyRunner) disableStrategy() {
	// TODO disable the strategy
	// TODO Stop the strategy
}
