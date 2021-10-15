package trigger

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Trigger interface {
	GetTriggerType() string
	GetPrice(time.Time) decimal.Decimal
	GetOperator() string
	SetOperator(string)
	ReadjustPrice(decimal.Decimal, time.Time)
	UpdatePriceByPercent(decimal.Decimal)
	Clone() Trigger
}

func NewTrigger(data map[string]interface{}) (t Trigger, err error) {
	triggerType, ok := data["trigger_type"].(string)
	if !ok {
		err = errors.New("'trigger_type' is missing")
		return
	}

	switch triggerType {
	case "line":
		return newLine(data)
	case "limit":
		return newLimit(data)
	default:
		err = fmt.Errorf("trigger_type '%s' not supported", triggerType)
	}

	return
}

// TODO Not yet supported
func NewTriggers(data []interface{}) (ts []Trigger, err error) {
	for _, trigger := range data {
		var t Trigger
		t, err = NewTrigger(trigger.(map[string]interface{}))
		if err != nil {
			return
		}
		ts = append(ts, t)
	}
	return
}

func IsTriggeredBySingleTrigger(trigger Trigger, t time.Time, price decimal.Decimal) bool {
	// Prevent panic if trigger hasn't been set yet
	if trigger == nil {
		return false
	}

	baselinePrice := trigger.GetPrice(t)
	switch trigger.GetOperator() {
	case ">=":
		if price.GreaterThanOrEqual(baselinePrice) {
			return true
		}
	case "<=":
		if price.LessThanOrEqual(baselinePrice) {
			return true
		}
	}

	return false
}

// TODO Not yet supported
func IsTriggeredByMultipleTriggers(operator string, triggers []Trigger, t time.Time, price decimal.Decimal) bool {
	switch operator {
	case "AND":
		for _, trigger := range triggers {
			baselinePrice := trigger.GetPrice(t)

			switch trigger.GetOperator() {
			case ">=":
				if price.LessThanOrEqual(baselinePrice) {
					return false
				}
			case "<=":
				if price.GreaterThanOrEqual(baselinePrice) {
					return false
				}
			}
		}
		return true
	case "OR":
		for _, trigger := range triggers {
			baselinePrice := trigger.GetPrice(t)

			switch trigger.GetOperator() {
			case ">=":
				if price.GreaterThanOrEqual(baselinePrice) {
					return true
				}
			case "<=":
				if price.LessThanOrEqual(baselinePrice) {
					return true
				}
			}
		}
		return false
	}

	return false
}

func validateOperator(operator string) error {
	switch operator {
	case ">=", "<=":
		return nil
	}
	return fmt.Errorf("operator '%s' not supported", operator)
}
