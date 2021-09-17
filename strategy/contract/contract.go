package contract

import (
	"crypto-trading-bot-main/strategy/order"
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type Status int

const (
	// position status
	OPENED Status = 1
	CLOSED Status = 0
)

type Hooker interface {
	// EntryOrder
	EntryTriggered(*Contract, time.Time, decimal.Decimal) (decimal.Decimal, error, bool)
	StopLossTriggerCreated(*Contract) (error, bool)

	// StopLossOrder
	StopLossTriggered(*Contract, time.Time, decimal.Decimal) error
	EntryBaselineTriggerUpdated(*Contract)

	// TakeProfitOrder
	TakeProfitTriggered(*Contract, time.Time, decimal.Decimal) error

	// Order's trigger gets updated
	OrderTriggerUpdated(*Contract)

	// Status changed
	StatusChanged(*Contract)
}

type Contract struct {
	ContractDirection order.ContractDirection
	EntryType         string
	EntryOrder        order.Order
	TakeProfitOrder   order.Order
	StopLossOrder     order.Order

	// To check if contract has been opened
	Status Status

	// entry_type 'baseline' only
	// In order to avoid false breakouit next time, record the highest price and time during the life cycle of each attempt
	BreakoutPeak struct {
		Time  time.Time
		Price decimal.Decimal
	}

	hook Hooker
}

func NewContract(contractDirection order.ContractDirection, data map[string]interface{}) (c *Contract, err error) {
	c = &Contract{
		Status: CLOSED, // default
	}

	// contract direction
	if contractDirection != order.LONG && contractDirection != order.SHORT {
		err = fmt.Errorf("contract_direction '%d' not supported", contractDirection)
		return
	}
	c.ContractDirection = contractDirection

	entryType, ok := data["entry_type"].(string)
	if !ok {
		err = errors.New("'entry_type' is missing")
		return
	}
	if entryType != order.ENTRY_LIMIT && entryType != order.ENTRY_BASELINE {
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
	entryOrder, err := order.NewOrder(contractDirection, entryType, "entry", eo)
	if err != nil {
		return
	}
	c.EntryOrder = entryOrder

	// (optional) take-profit order
	var takeProfitOrder order.Order
	tpo, ok := data["take_profit_order"].(map[string]interface{})
	if ok {
		takeProfitOrder, err = order.NewOrder(contractDirection, entryType, "take_profit", tpo)
		if err != nil {
			return
		}
	}
	c.TakeProfitOrder = takeProfitOrder

	// (optional) stop-loss order
	var stopLossOrder order.Order
	slo, ok := data["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrder, err = order.NewOrder(contractDirection, entryType, "stop_loss", slo)
		if err != nil {
			return
		}
	}
	c.StopLossOrder = stopLossOrder

	return
}

func (c *Contract) SetHook(h Hooker) {
	c.hook = h
}

func (c *Contract) SetStatus(status Status) {
	c.Status = status
}

func (c *Contract) CheckPrice(t time.Time, p decimal.Decimal) (err error, halted bool) {
	switch c.Status {
	case CLOSED:
		// Check if entry order is triggered
		if c.EntryOrder.IsTriggered(t, p) {
			// If both of entry order and one of stop-loss and take-profit order get triggered, do nothing
			// Otherwise, the stop-loss or take-profit order will be triggered immediately after entry-order triggered
			if c.StopLossOrder != nil && c.StopLossOrder.IsTriggered(t, p) {
				return
			}
			if c.TakeProfitOrder != nil && c.TakeProfitOrder.IsTriggered(t, p) {
				return
			}

			// Entry order is triggered
			var entryPrice decimal.Decimal
			if entryPrice, err, halted = c.hook.EntryTriggered(c, t, p); err != nil || halted {
				return
			}

			c.Status = OPENED
			c.hook.StatusChanged(c)

			// Set stop-loss trigger & order
			if c.StopLossOrder != nil {
				switch c.EntryType {
				case order.ENTRY_LIMIT:
					if err, halted = c.hook.StopLossTriggerCreated(c); err != nil || halted {
						return
					}
				case order.ENTRY_BASELINE:
					// For entry_type 'baseline', stop-loss order will depend on entry price
					c.setStopLossTrigger(entryPrice)
					if err, halted = c.hook.StopLossTriggerCreated(c); err != nil || halted {
						return
					}

					// Record breakout peak
					if c.StopLossOrder.(*order.StopLoss).BaselineReadjustmentEnabled {
						// Set breakout peak because price is default '0', it casues a bug in Short position
						c.setBreakoutPeak(t, p)
					}
				}
			}

			// To avoid a certain scenario that entry and stop-loss orders are triggered in turn constantly
			// For example:
			//      - entry trigger: mark price <= 43000
			//      - stop-loss trigger: mark price  <= 42000
			// These 2 orders will be constantly triggered when the mark price fluctuates around 42000 above and below
			// Fix this issue by changing the operator of entry trigger
			if c.EntryOrder.(*order.Entry).FlipOperatorEnabled {
				c.EntryOrder.(*order.Entry).UpdateOperator(c.ContractDirection)
			}

			c.hook.OrderTriggerUpdated(c)
			return
		}
	case OPENED:
		if c.EntryType == order.ENTRY_BASELINE && c.StopLossOrder != nil && c.StopLossOrder.(*order.StopLoss).BaselineReadjustmentEnabled {
			c.recordBreakoutPeak(t, p)
		}

		// Check if stop-loss order is triggered
		if c.StopLossOrder != nil && c.StopLossOrder.IsTriggered(t, p) {
			// Stop-loss order is triggered
			if err = c.hook.StopLossTriggered(c, t, p); err != nil {
				return
			}

			c.Status = CLOSED
			c.hook.StatusChanged(c)

			if c.EntryType == order.ENTRY_BASELINE {
				// Reset stop-loss trigger so when the mark price goes above entry won't be affected by previous stop-loss trigger
				c.StopLossOrder.(*order.StopLoss).UnsetTrigger()

				if c.StopLossOrder.(*order.StopLoss).BaselineReadjustmentEnabled {
					c.readjustEntryBaseline()
					c.hook.EntryBaselineTriggerUpdated(c)
					c.resetBreakoutPeak()
				}
				c.hook.OrderTriggerUpdated(c)
			}
			return
		}

		// Check take-profit order
		if c.TakeProfitOrder != nil && c.TakeProfitOrder.IsTriggered(t, p) {
			// Take-profit order is triggered
			if err = c.hook.TakeProfitTriggered(c, t, p); err != nil {
				return
			}

			c.Status = CLOSED
			c.hook.StatusChanged(c)
			halted = true
			return
		}
	}
	return
}

// entry_type 'baseline' only
// Set baseline price as cost price
func (c *Contract) setStopLossTrigger(p decimal.Decimal) {
	c.StopLossOrder.(*order.StopLoss).UpdateTriggerByLossPercent(c.ContractDirection, p)
}

// entry_type 'baseline' only
// Update baseline trigger and entry order for preventing false breakout
func (c *Contract) readjustEntryBaseline() {
	// Update baseline trigger first
	c.EntryOrder.(*order.Entry).UpdateBaselineTrigger(c.ContractDirection, c.BreakoutPeak.Price, c.BreakoutPeak.Time)

	// Update entry order based on baseline trigger and offset
	c.EntryOrder.(*order.Entry).UpdateTriggerByBaselineAndOffset(c.ContractDirection)
}

// entry_type 'baseline' only
func (c *Contract) setBreakoutPeak(t time.Time, p decimal.Decimal) {
	c.BreakoutPeak.Time = t
	c.BreakoutPeak.Price = p
}

// entry_type 'baseline' only
func (c *Contract) recordBreakoutPeak(t time.Time, p decimal.Decimal) {
	switch c.ContractDirection {
	case order.LONG:
		if p.GreaterThanOrEqual(c.BreakoutPeak.Price) {
			c.BreakoutPeak.Time = t
			c.BreakoutPeak.Price = p
		}
	case order.SHORT:
		if p.LessThanOrEqual(c.BreakoutPeak.Price) {
			c.BreakoutPeak.Time = t
			c.BreakoutPeak.Price = p
		}
	}
}

// entry_type 'baseline' only
func (c *Contract) resetBreakoutPeak() {
	c.BreakoutPeak.Time = time.Time{}
	c.BreakoutPeak.Price = decimal.Decimal{}
}
