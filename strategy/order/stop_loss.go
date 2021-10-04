package order

import (
	"crypto-trading-bot-engine/strategy/trigger"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type StopLoss struct {
	Trigger                     trigger.Trigger `json:"trigger,omitempty"`
	BaselineReadjustmentEnabled bool            `json:"baseline_readjustment_enabled"` // NOTE DO NOT 'omitempty' as you would be ignored when 'ParamsUpdated' tries to write into to DB
	LossTolerancePercent        float64         `json:"loss_tolerance_percent"`        // NOTE DO NOT 'omitempty' as you would be ignored when 'ParamsUpdated' tries to write into to DB
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
		// NOTE Context:
		//      Originally, for contract status 'CLOSED', only entry_type 'limit' needs to new 'Trigger'
		//      , as the 'Trigger' of 'baseline' will be set in runtime during 'contract.CheckPrice'
		//      When the runner restarts, it needs to get set with the 'Trigger' data from DB.
		//      In order to reduce the complexity, there is no check for distinguishing between 'OPENED' or 'CLOSED'
		//      and an extra work to refill 'Trigger' when contract status is 'OPENED'.
		//      'Trigger' will always be set anyway as long as it exists
		//      , it won't cause any issues for 'baseline' as the 'Trigger' will be overridden by 'contract.CheckPrice.setStopLossTrigger' when Entry triggered
		t, ok := data["trigger"].(map[string]interface{})
		if ok {
			var tt trigger.Trigger
			tt, err = trigger.NewTrigger(t)
			if err != nil {
				return &o, err
			}
			o.Trigger = tt
		}

		var p float64
		p, ok = data["loss_tolerance_percent"].(float64)
		if !ok {
			return &o, errors.New("'loss_tolerance_percent' is missing")
		}
		if p < 0 {
			return &o, errors.New("'loss_tolerance_percent' must be greater than or equal to 0")
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

func (o *StopLoss) UpdateTriggerByLossPercent(side Side, baselinePrice decimal.Decimal) {
	var t trigger.Trigger
	switch side {
	case LONG:
		// TODO new Limit trigger
		t = &trigger.Limit{
			TriggerType: "limit",
			Operator:    "<=",
			Price:       baselinePrice.Mul(decimal.NewFromFloat(1 - o.LossTolerancePercent)),
		}
	case SHORT:
		t = &trigger.Limit{
			TriggerType: "limit",
			Operator:    ">=",
			Price:       baselinePrice.Mul(decimal.NewFromFloat(1 + o.LossTolerancePercent)),
		}
	}
	o.Trigger = t
}
