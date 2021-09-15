package order

import (
	"crypto-trading-bot-main/strategy/trigger"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

const (
	ENTRY_LIMIT    = "limit"    // entry trigger is Limit trigger
	ENTRY_BASELINE = "baseline" // entry trigger (Line trigger) and stop-loss trigger (Limit trigger) are based on baseline
)

type Order interface {
	IsTriggered(time.Time, decimal.Decimal) bool
	GetTrigger() trigger.Trigger
	SetTrigger(trigger.Trigger)
}

func NewOrder(positionType string, entryType, orderType string, data map[string]interface{}) (o Order, err error) {
	switch orderType {
	case "entry":
		return NewEntry(positionType, entryType, data)
	case "take_profit":
		return NewTakeProfit(data)
	case "stop_loss":
		return NewStopLoss(entryType, data)
	default:
		err = fmt.Errorf("order type '%s' not supported", orderType)
	}
	return
}
