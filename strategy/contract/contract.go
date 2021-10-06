package contract

import (
	"crypto-trading-bot-engine/strategy/order"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Status int64

const (
	// position status
	CLOSED  Status = 0
	OPENED  Status = 1
	UNKNOWN Status = 2

	BREAKOUT_PEAK_TRIGGERED_INTERVAL = 20 // second
)

type Mark struct {
	Price decimal.Decimal
	Time  time.Time
}

type Hooker interface {
	// EntryOrder
	EntryTriggered(*Contract, time.Time, decimal.Decimal) (decimal.Decimal, bool, error)
	StopLossTriggerCreated(*Contract) (bool, error)

	// StopLossOrder
	StopLossTriggered(*Contract) (bool, error)
	EntryTrendlineTriggerUpdated(*Contract)
	EntryTriggerOperatorUpdated(*Contract)

	// TakeProfitOrder
	TakeProfitTriggered(*Contract) error

	// Entry order trigger gets updated
	ParamsUpdated(*Contract) (bool, error)

	// TODO test
	// Breakout peak updated after cooldown
	BreakoutPeakUpdated(*Contract)
}

type Contract struct {
	Side            order.Side
	EntryType       string
	EntryOrder      order.Order
	TakeProfitOrder order.Order
	StopLossOrder   order.Order

	// The status of the contract
	Status Status

	// entry_type 'trendline' only
	// In order to avoid false breakouit next time, record the highest price and time during the life cycle of each attempt
	BreakoutPeak struct {
		Time  time.Time
		Price decimal.Decimal

		// Trigger a funtion after a period of cooldown time when it gets updated
		lastTriggeredTime time.Time // the last triggered time
	}

	hook Hooker
}

func NewContract(side order.Side, data map[string]interface{}) (c *Contract, err error) {
	c = &Contract{
		Status: CLOSED, // default
	}

	// contract direction
	if side != order.LONG && side != order.SHORT {
		err = fmt.Errorf("side '%d' not supported", side)
		return
	}
	c.Side = side

	entryType, ok := data["entry_type"].(string)
	if !ok {
		err = errors.New("'entry_type' is missing")
		return
	}
	if entryType != order.ENTRY_LIMIT && entryType != order.ENTRY_TRENDLINE {
		err = fmt.Errorf("entry_type '%s' not supported", entryType)
		return
	}
	c.EntryType = entryType

	// entry order
	eo, ok := data["entry_order"].(map[string]interface{})
	if !ok {
		err = errors.New("'entry_order' is missing")
		return
	}
	entryOrder, err := order.NewOrder(side, entryType, "entry", eo)
	if err != nil {
		return
	}
	c.EntryOrder = entryOrder

	// (optional) take-profit order
	var takeProfitOrder order.Order
	tpo, ok := data["take_profit_order"].(map[string]interface{})
	if ok {
		takeProfitOrder, err = order.NewOrder(side, entryType, "take_profit", tpo)
		if err != nil {
			return
		}
	}
	c.TakeProfitOrder = takeProfitOrder

	// (optional) stop-loss order
	var stopLossOrder order.Order
	slo, ok := data["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrder, err = order.NewOrder(side, entryType, "stop_loss", slo)
		if err != nil {
			return
		}
	}
	c.StopLossOrder = stopLossOrder

	// Breakout peak
	bp, ok := data["breakout_peak"].(map[string]interface{})
	if ok {
		// time
		t, ok := bp["time"].(string)
		if !ok {
			return c, errors.New("'time' is missing")
		}
		tt, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return c, fmt.Errorf("failed to parse 'time', err: %v", err)
		}

		// price
		p, ok := bp["price"].(string)
		if !ok {
			return c, errors.New("'price' is missing or not string")
		}
		pp, err := decimal.NewFromString(p)
		if err != nil {
			return c, errors.New("'price' isn't a stringified number")
		}

		c.BreakoutPeak.Time = tt
		c.BreakoutPeak.Price = pp
	}

	return
}

func (c *Contract) SetHook(h Hooker) {
	c.hook = h
}

func (c *Contract) SetStatus(status Status) {
	c.Status = status
}

func (c *Contract) CheckPrice(mark Mark) (halted bool, err error) {
	switch c.Status {
	case CLOSED:
		// Check if entry order is triggered
		if c.EntryOrder.IsTriggered(mark.Time, mark.Price) {
			// If both of entry order and one of stop-loss and take-profit order get triggered, do nothing
			// Otherwise, the stop-loss or take-profit order will be triggered immediately after entry-order triggered
			if c.StopLossOrder != nil && c.StopLossOrder.IsTriggered(mark.Time, mark.Price) {
				return
			}
			if c.TakeProfitOrder != nil && c.TakeProfitOrder.IsTriggered(mark.Time, mark.Price) {
				return
			}

			// Entry order is triggered
			var entryPrice decimal.Decimal
			if entryPrice, halted, err = c.hook.EntryTriggered(c, mark.Time, mark.Price); err != nil || halted {
				return
			}
			c.Status = OPENED

			// Set stop-loss trigger & order
			if c.StopLossOrder != nil {
				switch c.EntryType {
				case order.ENTRY_LIMIT:
					if halted, err = c.hook.StopLossTriggerCreated(c); err != nil || halted {
						return
					}
				case order.ENTRY_TRENDLINE:
					// For entry_type 'trendline', stop-loss order will depend on entry price
					c.setStopLossTrigger(entryPrice)
					if halted, err = c.hook.StopLossTriggerCreated(c); err != nil || halted {
						return
					}

					// Record breakout peak
					if c.StopLossOrder.(*order.StopLoss).TrendlineReadjustmentEnabled {
						// Set breakout peak because price is default '0', it casues a bug in Short position
						c.setBreakoutPeak(mark.Time, mark.Price)
					}
				}
			}

			// To avoid a certain scenario that entry and stop-loss orders are triggered in turn constantly
			// For example:
			//      - entry trigger: mark price <= 43000
			//      - stop-loss trigger: mark price  <= 42000
			//			These 2 orders will be constantly triggered when the mark price fluctuates around 42000 above and below
			//			Fix this issue by changing the operator of entry trigger
			if c.EntryOrder.(*order.Entry).FlipOperatorEnabled {
				c.EntryOrder.(*order.Entry).FlipOperator(c.Side)
				c.hook.EntryTriggerOperatorUpdated(c)
				c.EntryOrder.(*order.Entry).FlipOperatorEnabled = false
			}

			// For entry trigger during initialisation and setStopLossTrigger
			if halted, err = c.hook.ParamsUpdated(c); err != nil || halted {
				return
			}

			return
		}
	case OPENED:
		if c.EntryType == order.ENTRY_TRENDLINE && c.StopLossOrder != nil && c.StopLossOrder.(*order.StopLoss).TrendlineReadjustmentEnabled {
			if c.recordBreakoutPeak(mark.Time, mark.Price) {
				// If the breakout has been updated, trigger the function after cooldown
				if mark.Time.After(c.BreakoutPeak.lastTriggeredTime.Add(time.Second * time.Duration(BREAKOUT_PEAK_TRIGGERED_INTERVAL))) {
					c.hook.BreakoutPeakUpdated(c)
					c.BreakoutPeak.lastTriggeredTime = mark.Time
				}
			}
		}

		// Check if stop-loss order is triggered
		if c.StopLossOrder != nil && c.StopLossOrder.IsTriggered(mark.Time, mark.Price) {
			// Stop-loss order is triggered
			if halted, err = c.hook.StopLossTriggered(c); err != nil || halted {
				return
			}
			c.Status = CLOSED

			if c.EntryType == order.ENTRY_TRENDLINE {
				// Reset stop-loss trigger so when the mark price goes above entry won't be affected by previous stop-loss trigger
				c.StopLossOrder.(*order.StopLoss).UnsetTrigger()

				if c.StopLossOrder.(*order.StopLoss).TrendlineReadjustmentEnabled {
					c.readjustEntryTrendline()
					c.hook.EntryTrendlineTriggerUpdated(c)
					c.resetBreakoutPeak()
				}
			}

			// For readjustEntryTrendline and stop-loss UnsetTrigger
			if halted, err = c.hook.ParamsUpdated(c); err != nil || halted {
				return
			}

			return
		}

		// Check take-profit order
		if c.TakeProfitOrder != nil && c.TakeProfitOrder.IsTriggered(mark.Time, mark.Price) {
			// Take-profit order is triggered
			c.Status = CLOSED
			err = c.hook.TakeProfitTriggered(c)
			halted = true

			return
		}
	case UNKNOWN:
		return true, errors.New("unknown status")
	}
	return
}

// entry_type 'trendline' only
// Set trendline price as cost price
func (c *Contract) setStopLossTrigger(p decimal.Decimal) {
	c.StopLossOrder.(*order.StopLoss).UpdateTriggerByLossPercent(c.Side, p)
}

// entry_type 'trendline' only
// Update trendline trigger and entry order for preventing false breakout
func (c *Contract) readjustEntryTrendline() {
	// Update trendline trigger first
	c.EntryOrder.(*order.Entry).UpdateTrendlineTrigger(c.Side, c.BreakoutPeak.Price, c.BreakoutPeak.Time)

	// Update trigger based on trendline trigger and offset
	c.EntryOrder.(*order.Entry).UpdateTriggerByTrendlineAndOffset(c.Side)
}

// entry_type 'trendline' only
func (c *Contract) setBreakoutPeak(t time.Time, p decimal.Decimal) {
	c.BreakoutPeak.Time = t
	c.BreakoutPeak.Price = p
}

// entry_type 'trendline' only
func (c *Contract) recordBreakoutPeak(t time.Time, p decimal.Decimal) bool {
	updated := false
	switch c.Side {
	case order.LONG:
		if p.GreaterThanOrEqual(c.BreakoutPeak.Price) {
			c.BreakoutPeak.Time = t
			c.BreakoutPeak.Price = p
			updated = true
		}
	case order.SHORT:
		if p.LessThanOrEqual(c.BreakoutPeak.Price) {
			c.BreakoutPeak.Time = t
			c.BreakoutPeak.Price = p
			updated = true
		}
	}
	return updated
}

// entry_type 'trendline' only
func (c *Contract) resetBreakoutPeak() {
	c.BreakoutPeak.Time = time.Time{}
	c.BreakoutPeak.Price = decimal.Decimal{}
}

// TODO test
func TranslateStatus(s Status) string {
	switch s {
	case CLOSED:
		return "Closed"
	case OPENED:
		return "Opened"
	case UNKNOWN:
		return "Unknown"
	}
	return ""
}

// TODO test
func TranslateStatusByInt(s int64) string {
	return TranslateStatus(Status(s))
}
