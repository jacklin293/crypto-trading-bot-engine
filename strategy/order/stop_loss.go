package order

import (
	"crypto-trading-bot-main/strategy/trigger"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type StopLoss struct {
	Trigger                     trigger.Trigger
	BaselineReadjustmentEnabled bool
	LossTolerancePercent        float64
}

func NewStopLoss(entryType string, data map[string]interface{}) (*StopLoss, error) {
	var o StopLoss
	var err error

	switch entryType {
	case ENTRY_LIMIT:
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
	case ENTRY_BASELINE:
		var ok bool
		var p float64
		p, ok = data["loss_tolerance_percent"].(float64)
		if !ok {
			return &o, errors.New("'loss_tolerance_percent' is missing")
		}
		if p <= 0 {
			return &o, errors.New("'loss_tolerance_percent' must be greater than 0")
		}
		o.LossTolerancePercent = p

		var enabled bool
		enabled, ok = data["baseline_readjustment_enabled"].(bool)
		if ok {
			o.BaselineReadjustmentEnabled = enabled
		}
	}

	return &o, err
}

func (o *StopLoss) GetTrigger() trigger.Trigger {
	return o.Trigger
}

func (o *StopLoss) SetTrigger(source trigger.Trigger) {
	newTrigger := source.Clone()
	o.Trigger = newTrigger
}

func (o *StopLoss) UnsetTrigger() {
	o.Trigger = nil
}

func (o *StopLoss) IsTriggered(t time.Time, p decimal.Decimal) bool {
	return trigger.IsTriggeredBySingleTrigger(o.Trigger, t, p)
}

func (o *StopLoss) UpdateTriggerByLossPercent(contractDirection ContractDirection, baselinePrice decimal.Decimal) {
	var t trigger.Trigger
	switch contractDirection {
	case LONG:
		t = &trigger.Limit{
			Operator: "<=",
			Price:    baselinePrice.Mul(decimal.NewFromFloat(1 - o.LossTolerancePercent)),
		}
	case SHORT:
		t = &trigger.Limit{
			Operator: ">=",
			Price:    baselinePrice.Mul(decimal.NewFromFloat(1 + o.LossTolerancePercent)),
		}
	}
	o.Trigger = t
}
