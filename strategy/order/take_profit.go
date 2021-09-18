package order

import (
	"crypto-trading-bot-main/strategy/trigger"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type TakeProfit struct {
	Trigger trigger.Trigger `json:"trigger,omitempty"`
}

func NewTakeProfit(data map[string]interface{}) (*TakeProfit, error) {
	var o TakeProfit
	var err error

	t, ok := data["trigger"].(map[string]interface{})
	if !ok {
		return &o, errors.New("'trigger' is missing")
	}
	var tt trigger.Trigger
	tt, err = trigger.NewTrigger(t)
	if err != nil {
		return &o, err
	}
	o.Trigger = tt

	return &o, err
}

func (o *TakeProfit) GetTrigger() trigger.Trigger {
	return o.Trigger
}

func (o *TakeProfit) SetTrigger(source trigger.Trigger) {
	newTrigger := source.Clone()
	o.Trigger = newTrigger
}

func (o *TakeProfit) IsTriggered(t time.Time, p decimal.Decimal) bool {
	return trigger.IsTriggeredBySingleTrigger(o.Trigger, t, p)
}
