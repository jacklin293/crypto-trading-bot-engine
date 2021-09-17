package strategy

import (
	"crypto-trading-bot-main/strategy/contract"
	"fmt"
	"log"
	"time"

	"github.com/shopspring/decimal"
)

// For storing the func names that are triggered and being used to compare with expected results
type contractHook struct {
	logger *log.Logger

	strategy *Strategy // TODO?
}

func newContractHook() *contractHook {
	return &contractHook{}
}

func (ch *contractHook) setLogger(l *log.Logger) {
	ch.logger = l
}

func (ch *contractHook) EntryTriggered(c *contract.Contract, t time.Time, p decimal.Decimal) (decimal.Decimal, error, bool) {
	// TODO LOCK   key['strategy_id']['symbol']
	// TODO check LOCK each time

	// TODO switch LONG/SHORT
	// TODO Send the order
	// TODO telegram

	ch.logger.Println("EntryTriggered")

	return p, nil, false
}

func (ch *contractHook) StopLossTriggerCreated(c *contract.Contract) (error, bool) {
	ch.logger.Println("StopLossTriggerCreated")
	// TODO if failed, cancel close position
	// TODO telegram

	return nil, false
}

func (ch *contractHook) StopLossTriggered(c *contract.Contract, t time.Time, p decimal.Decimal) error {
	fmt.Println("StopLossTriggered")
	// TODO telegram

	return nil
}

func (ch *contractHook) EntryBaselineTriggerUpdated(c *contract.Contract) {
	fmt.Println("EntryBaselineTriggerUpdated")
	// TODO telegram
}

func (ch *contractHook) TakeProfitTriggered(c *contract.Contract, t time.Time, p decimal.Decimal) error {
	fmt.Println("TakeProfitTriggered")
	// TODO send order
	// TODO telegram
	return nil
}

func (ch *contractHook) OrderTriggerUpdated(c *contract.Contract) {
	fmt.Println("OrderTriggerUpdated")
	// TODO update DB
}

func (ch *contractHook) StatusChanged(c *contract.Contract) {
	fmt.Println("StatusChanged")
	// TODO update DB
	// TODO telegram
}
