package order

import (
	"crypto-trading-bot-engine/strategy/trigger"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

type Entry struct {
	Trigger               trigger.Trigger `json:"trigger,omitempty"`
	BaselineTrigger       trigger.Trigger `json:"baseline_trigger,omitempty"`
	BaselineOffsetPercent float64         `json:"baseline_offset_percent,omitempty"`
	FlipOperatorEnabled   bool            `json:"flip_operator_enabled,omitempty"`
}

func NewEntry(side Side, entryType string, data map[string]interface{}) (*Entry, error) {
	var o Entry
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
		// baseline trigger
		bt, ok := data["baseline_trigger"].(map[string]interface{})
		if !ok {
			return &o, errors.New("'baseline_trigger' is missing")
		}
		var tt trigger.Trigger
		tt, err = trigger.NewTrigger(bt)
		if err != nil {
			return &o, err
		}
		o.BaselineTrigger = tt

		// baseline_offset_percent
		// NOTE For contract status `OPENED`, there is no extra work to refill `Trigger` as `UpdateTriggerByBaselineAndOffset` has done the job
		var p float64
		p, ok = data["baseline_offset_percent"].(float64)
		if !ok {
			return &o, errors.New("'baseline_offset_percent' is missing")
		}
		o.BaselineOffsetPercent = p
		o.UpdateTriggerByBaselineAndOffset(side)
	}

	var enabled bool
	enabled, ok := data["flip_operator_enabled"].(bool)
	if ok {
		o.FlipOperatorEnabled = enabled
	}

	return &o, err
}

func (o *Entry) GetTrigger() trigger.Trigger {
	return o.Trigger
}

func (o *Entry) SetTrigger(source trigger.Trigger) {
	newTrigger := source.Clone()
	o.Trigger = newTrigger
}

func (o *Entry) IsTriggered(t time.Time, p decimal.Decimal) bool {
	return trigger.IsTriggeredBySingleTrigger(o.Trigger, t, p)
}

// entry_type 'baseline' only
func (o *Entry) UpdateBaselineTrigger(side Side, p2 decimal.Decimal, t2 time.Time) {
	// If trigger type is Limit, set the price given
	// If trigger type is Line, when price2 > price1, set price2 = price1
	lineTrigger, ok := o.BaselineTrigger.(*trigger.Line)
	if ok {
		switch side {
		case LONG:
			if p2.GreaterThanOrEqual(lineTrigger.Price1) {
				p2 = lineTrigger.Price1
			}
		case SHORT:
			if p2.LessThanOrEqual(lineTrigger.Price1) {
				p2 = lineTrigger.Price1
			}
		}
	}
	o.BaselineTrigger.ReadjustPrice(p2, t2)
}

// entry_type 'baseline' only
// For long position, entry order will be triggered at the price higher than baseline
// For short position, entry order will be triggered at the price lower than baseline
func (o *Entry) UpdateTriggerByBaselineAndOffset(side Side) {
	// entry order based on baseline_trigger and offset percent
	var percent decimal.Decimal
	switch side {
	case LONG:
		percent = decimal.NewFromFloat(1 + o.BaselineOffsetPercent)
	case SHORT:
		percent = decimal.NewFromFloat(1 - o.BaselineOffsetPercent)
	}
	// Use SetTrigger to prevent BaselineTrigger from being modified due to pointer
	o.SetTrigger(o.BaselineTrigger)
	o.Trigger.UpdatePriceByPercent(percent)
}

// Flip operator
func (o *Entry) FlipOperator(side Side) {
	switch side {
	case LONG:
		// Trigger always exists, but just in case
		if o.Trigger != nil {
			o.Trigger.SetOperator(">=")
		}

		// entry_type 'limit' doesn't have BaselineTrigger
		if o.BaselineTrigger != nil {
			o.BaselineTrigger.SetOperator(">=")
		}
	case SHORT:
		// Trigger always exists, but just in case
		if o.Trigger != nil {
			o.Trigger.SetOperator("<=")
		}

		// entry_type 'limit' doesn't have BaselineTrigger
		if o.BaselineTrigger != nil {
			o.BaselineTrigger.SetOperator("<=")
		}
	}
}
