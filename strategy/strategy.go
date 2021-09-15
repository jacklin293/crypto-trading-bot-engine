package strategy

import (
	"log"
	"sync"
	"time"

	"crypto-trading-bot-main/strategy/contract"

	"github.com/shopspring/decimal"
)

type Mark struct {
	Price decimal.Decimal
	Time  time.Time
}

type Strategy struct {
	id           string // id from db
	symbol       string
	positionType string
	logger       *log.Logger
	StopCh       chan bool
	MarkCh       chan Mark

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
func NewStrategy(id string, symbol string, positionType string, status contract.Status, params map[string]interface{}) (*Strategy, error) {
	ch := newContractHook()
	c, err := contract.NewContract(positionType, params)
	if err != nil {
		return &Strategy{}, err
	}
	c.SetHook(ch)
	c.SetStatus(status)

	s := &Strategy{
		id:           id,
		symbol:       symbol,
		positionType: positionType,
		contract:     c,
		contractHook: ch,
		StopCh:       make(chan bool),
		MarkCh:       make(chan Mark),
	}

	return s, err
}

func (s *Strategy) SetLogger(l *log.Logger) {
	s.logger = l
	s.contractHook.setLogger(s.logger)
}

func (s *Strategy) SetBeforeCloseFunc(f func(string, string)) {
	s.beforeCloseFunc = f
}

func (s *Strategy) SetHandlerBlockWg(wg *sync.WaitGroup) {
	s.handlerBlockWg = wg
}

// Strategy
func (s *Strategy) Run() {
	halted := false
	for {
		select {
		case <-s.StopCh:
			halted = true
			break
		case mark := <-s.MarkCh:
			// If 'CheckPrice' is still in progress, ignore the incoming prices unitl it's finished
			if s.ignoreIncomingMark {
				break
			}
			s.ignoreIncomingMark = true
			go s.checkPrice(&mark)
		}
		if halted {
			break
		}
	}

	// This might not be necessary, but better to wait until removal process done
	s.handlerBlockWg.Add(1)
	defer s.handlerBlockWg.Done()
	s.beforeCloseFunc(s.symbol, s.id)
}

func (s *Strategy) checkPrice(mark *Mark) {
	// Prevent the process from being closed when anything is in progress
	s.handlerBlockWg.Add(1)
	defer s.handlerBlockWg.Done()

	// TODO del
	s.logger.Println(s.id, mark.Time.Format("2006-01-02 15:04:05"), mark.Price)

	s.contract.CheckPrice(mark.Time, mark.Price)
	s.ignoreIncomingMark = false
}
