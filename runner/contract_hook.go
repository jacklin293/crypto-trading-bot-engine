package runner

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"crypto-trading-bot-engine/message"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
	"crypto-trading-bot-engine/strategy/trigger"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// For storing the func names that are triggered and being used to compare with expected results
type contractHook struct {
	contractStrategy *db.ContractStrategy
	user             *db.User

	logger   *log.Logger
	db       *db.DB
	exchange exchange.Exchanger // by user_id

	// Send the notification, only support telegram atm
	sender message.Messenger // all users use the same one, but sent with different chat_id

	// Check if entry order by symbol has been triggered already
	symbolEntryTakenMutex map[string]*sync.Mutex
}

func newContractHook(cs *db.ContractStrategy) *contractHook {
	return &contractHook{
		contractStrategy: cs,
	}
}

func (ch *contractHook) setLogger(l *log.Logger) {
	ch.logger = l
}

func (ch *contractHook) setDB(db *db.DB) {
	ch.db = db
}

func (ch *contractHook) setSymbolEntryTakenMutex(m map[string]*sync.Mutex) {
	ch.symbolEntryTakenMutex = m
}

func (ch *contractHook) setExchange(ex exchange.Exchanger) {
	ch.exchange = ex
}

func (ch *contractHook) setSender(m message.Messenger) {
	ch.sender = m
}

func (ch *contractHook) setUser(u *db.User) {
	ch.user = u
}

func (ch *contractHook) EntryTriggered(c *contract.Contract, t time.Time, p decimal.Decimal) (decimal.Decimal, bool, error) {
	// Make sure only one order by symbol can be triggered at once
	// Also, from FTX doc: One websocket connection may be logged in to at most one user.
	mutex := ch.symbolEntryTakenMutex[ch.contractStrategy.UserUuid]
	mutex.Lock()
	defer mutex.Unlock()

	var text string

	// Check if the strategy conflicts with other strategy with the same symbol
	// Uuid should be excluded, otherwise it can get the row of itself as status has been changed to 'opening' before entry triggered
	_, count, err := ch.db.GetNonClosedContractStrategiesBySymbol(ch.contractStrategy.UserUuid, ch.contractStrategy.Symbol, ch.contractStrategy.Uuid)
	if err != nil {
		return p, false, fmt.Errorf("EntryTriggered - failed to get non-closed contract strategies, err: %v", err)
	}
	if count > 0 {
		text := fmt.Sprintf("[Warn] '%s %s $%s' is triggered, but conflicts with others. Please make sure other %s strategies' status are 'closed'. This strategy will be reset shortly", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, ch.contractStrategy.Margin, ch.contractStrategy.Symbol)
		ch.notify(text)
		return p, true, nil
	}

	// Calculate the size
	size := ch.contractStrategy.Margin.DivRound(p, 8)

	// Place entry order
	orderId, err := ch.exchange.PlaceEntryOrder(ch.contractStrategy.Symbol, order.Side(ch.contractStrategy.Side), size)
	if err != nil {
		text := fmt.Sprintf("[Error] Failed to place entry order, err: %v", err)
		ch.notify(text)
		return p, false, fmt.Errorf("EntryTriggered - failed to place entry order, err: %v", err)
	}

	// Notification
	text = fmt.Sprintf("[Entry] '%s %s $%s' has been triggered @%s", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, ch.contractStrategy.Margin.StringFixed(0), p.String())
	ch.notify(text)

	// For memory data
	ch.contractStrategy.PositionStatus = int64(contract.OPENED)
	ch.contractStrategy.ExchangeOrdersDetails = datatypes.JSONMap{
		"entry_order": map[string]interface{}{
			"order_id": float64(orderId),
			"price":    p.String(),
			"size":     size.String(),
		},
	}
	ch.contractStrategy.LastPositionAt = time.Now()

	// For DB
	contractStrategy := map[string]interface{}{
		"position_status":         ch.contractStrategy.PositionStatus,
		"exchange_orders_details": ch.contractStrategy.ExchangeOrdersDetails,
		"last_position_at":        ch.contractStrategy.LastPositionAt,
	}
	_, err = ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy)
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return p, true, fmt.Errorf("EntryTriggered - failed to update 'exchange_orders_details', err: %v", err)
	}
	return p, false, nil
}

func (ch *contractHook) StopLossTriggerCreated(c *contract.Contract) (bool, error) {
	var text string

	// entry_type 'limit' and 'trendline' both are using Limit Trigger, time doesn't matter
	p := c.StopLossOrder.(*order.StopLoss).Trigger.GetPrice(time.Now())
	size, err := decimal.NewFromString(ch.contractStrategy.ExchangeOrdersDetails["entry_order"].(map[string]interface{})["size"].(string))
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("StopLossTriggerCreated - failed to convert 'size' from order info, err: %v", err)
	}

	// Place stop-loss order - retyr 30 times, interval 2 secs
	orderId, err := ch.exchange.RetryPlaceStopLossOrder(ch.contractStrategy.Symbol, order.Side(ch.contractStrategy.Side), p, size, 30, 2)
	if err != nil {
		text = fmt.Sprintf("[Error] %s %s - failed to place stop-loss order, err: %v", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, err)
		ch.notify(text)
		ch.closePosition()
		return true, fmt.Errorf("StopLossTriggerCreated - failed to place stop-loss order, err: %v", err)
	}

	// Notification
	text = fmt.Sprintf("[Info] %s stop-loss order has been placed @%s", ch.contractStrategy.Symbol, p)
	ch.notify(text)

	// update memory data
	ch.contractStrategy.ExchangeOrdersDetails["stop_loss_order"] = map[string]interface{}{
		"order_id": float64(orderId), // make it more consistent by turning it into float64
	}
	// update db
	contractStrategy := map[string]interface{}{
		"exchange_orders_details": ch.contractStrategy.ExchangeOrdersDetails,
	}
	_, err = ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy)
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("StopLossTriggerCreated - failed to update 'exchange_order_details', err: %v", err)
	}

	return false, nil
}

func (ch *contractHook) StopLossTriggered(c *contract.Contract) (bool, error) {
	text := fmt.Sprintf("[Stop-loss] '%s %s' has been triggered", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)

	if err := ch.closePosition(); err != nil {
		return true, fmt.Errorf("StopLossTriggered - failed to close position, err: %v", err)
	}

	// Reset status and exchange_orders_details
	contractStrategy := map[string]interface{}{
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	_, err := ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy)
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("StopLossTriggered - failed to update 'position_status', err: %v", err)
	}

	// Update memory data
	ch.contractStrategy.PositionStatus = int64(contract.CLOSED)
	ch.contractStrategy.ExchangeOrdersDetails = datatypes.JSONMap{}
	return false, nil
}

func (ch *contractHook) EntryTrendlineTriggerUpdated(c *contract.Contract) {
	text := fmt.Sprintf("[Info] '%s %s' entry trend line has been updated", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)

	// Send new trendline
	t := c.EntryOrder.(*order.Entry).TrendlineTrigger
	// trigger shouldn't be 'nil', but just in case that it won't blow up
	if t != nil {
		p1 := t.(*trigger.Line).Price1
		t1 := t.(*trigger.Line).Time1
		p2 := t.(*trigger.Line).Price2
		t2 := t.(*trigger.Line).Time2
		text = fmt.Sprintf("[Info] New entry trend line:\nPoint 1: $%s, '%s'\nPoint 2: $%s, '%s'", p1, t1.Format("2006-01-02 15:04"), p2, t2.Format("2006-01-02 15:04"))
		ch.notify(text)
	}
}

func (ch *contractHook) EntryTriggerOperatorUpdated(c *contract.Contract) {
	text := fmt.Sprintf("[Info] '%s %s' entry trigger operator has been updated", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)
}

// NOTE Take-profit will always halt the strategy regardless of whether err is thrown
func (ch *contractHook) TakeProfitTriggered(c *contract.Contract) error {
	text := fmt.Sprintf("[Take-profit] '%s %s' has been triggered", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)

	// Update memory data
	ch.contractStrategy.Enabled = 0

	// NOTE DB data will be updated via event channel
	return ch.closePosition()
}

// NOTE datatypes.JSONMap will escapte `<` into `\u003c`, but it's fine. It can still be unmarchal and turned back to `=` without issue
// NOTE datatypes.JSONMap will turm time into `2021-09-15T04:00:00Z`
// NOTE For entry_type 'limit', will have some params that shouldn't have had after this update like `trendline_offset_percent` and `loss_tolerance_percent`, but it's fine
func (ch *contractHook) ParamsUpdated(c *contract.Contract) (bool, error) {
	// NOTE Don't save `breakout_peak`, because we want it to be reset after stop-loss order triggered
	// Update memory data
	ch.contractStrategy.Params = datatypes.JSONMap{
		"entry_type":  c.EntryType,
		"entry_order": c.EntryOrder,
	}
	if c.StopLossOrder != nil {
		ch.contractStrategy.Params["stop_loss_order"] = c.StopLossOrder
	}
	if c.TakeProfitOrder != nil {
		ch.contractStrategy.Params["take_profit_order"] = c.TakeProfitOrder
	}

	// Update db
	contractStrategy := map[string]interface{}{
		"params": ch.contractStrategy.Params,
	}
	if _, err := ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy); err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("ParamsUpdated - failed to update 'params', err: %v", err)
	}

	return false, nil
}

// NOTE We don't dont to worry about reset process
// NOTE When entry triggered, regardless of what breakout peak is, it will be overriden by `setBreakoutPeak`
// NOTE When stop-loss triggered, it reset breakout peak, and trigger `ParamsUpdated` at the end, which doesn't write
//      `breakout peak` into DB
// NOTE Because of cooldown period, the real breakout peak might not be the same as breakout peak in memory
//      , as checkPrice is still running and update the value, but it's fine, not a big deal
func (ch *contractHook) BreakoutPeakUpdated(c *contract.Contract) {
	// NOTE for debug
	text := fmt.Sprintf("breakout peak {price: %s, time: %s}", c.BreakoutPeak.Price, c.BreakoutPeak.Time.Format("2006-01-02 15:04:05"))
	ch.logger.Printf("[Debug] sid: %s uid: %s sym: %s text: '%s'", ch.contractStrategy.Uuid, ch.contractStrategy.UserUuid, ch.contractStrategy.Symbol, text)

	// Update memory data
	ch.contractStrategy.Params["breakout_peak"] = map[string]interface{}{
		"time":  c.BreakoutPeak.Time,
		"price": c.BreakoutPeak.Price,
	}

	// Update db
	contractStrategy := map[string]interface{}{
		"params": ch.contractStrategy.Params,
	}
	if _, err := ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy); err != nil {
		ch.logger.Printf("[Error] failed to save breakout peak, err: %v", err)
	}
}

func (ch *contractHook) closePosition() error {
	var text string

	closedAlready, err := ch.closeOpenPosition()
	if err != nil {
		return fmt.Errorf("closePosition err: %v", err)
	}
	if closedAlready {
		text = fmt.Sprintf("[Info] '%s %s' position have been closed already", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return nil
	} else {
		// Notification
		text = fmt.Sprintf("[Info] '%s %s' position has been closed successfully", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
	}

	// Cancel stop-loss order if exists
	// It's possible that there is no order_id for stop-loss order that will happen when the position has been opened but something went wrong before stop-loss order is created
	orderInfo, ok := ch.contractStrategy.ExchangeOrdersDetails["stop_loss_order"].(map[string]interface{})
	if ok {
		tmpId, ok := orderInfo["order_id"].(float64)
		if !ok {
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
			ch.notify(text)
			return fmt.Errorf("closePosition - stop_loss_order.order_id is missing")
		}
		stopLossOrderId := int64(tmpId)
		if err = ch.exchange.RetryCancelOpenTriggerOrder(stopLossOrderId, 20, 2); err != nil {
			text = fmt.Sprintf("[Error] Failed to cancel %s stop-loss order, err: %v", ch.contractStrategy.Symbol, err)
			ch.notify(text)
			return err
		}
	}

	// Update memory data
	ch.contractStrategy.PositionStatus = int64(contract.CLOSED)
	ch.contractStrategy.ExchangeOrdersDetails = datatypes.JSONMap{}
	return nil
}

// When closed is true, it means that it might have been closed by stop-loss trigger order by FTX
func (ch *contractHook) closeOpenPosition() (closed bool, err error) {
	// Get size from open position
	positionInfo, err := ch.exchange.RetryGetPosition(ch.contractStrategy.Symbol, 30, 2)
	if err != nil {
		return false, fmt.Errorf("closeOpenPosition - failed to get open position, err: %v", err)
	}

	// If size is zero, it means that it might be closed already
	if positionInfo["size"].(string) == "0" {
		return true, nil
	}
	size, err := decimal.NewFromString(positionInfo["size"].(string))
	if err != nil {
		return false, fmt.Errorf("closeOpenPosition - failed to convert size, err: %v", err)
	}

	// Close position
	if err := ch.exchange.RetryClosePosition(ch.contractStrategy.Symbol, order.Side(ch.contractStrategy.Side), size, 30, 2); err != nil {
		// position could be closed by stop-loss trigger order, it's fine for caller `StopLossTriggered`
		return false, fmt.Errorf("closeOpenPosition - failed to close position, err: %v", err)
	}

	return false, nil
}

func (ch *contractHook) notify(text string) {
	ch.logger.Printf("sid: %s uid: %s sym: %s text: '%s'", ch.contractStrategy.Uuid, ch.contractStrategy.UserUuid, ch.contractStrategy.Symbol, text)
	go ch.sender.Send(ch.user.TelegramChatId, text)
}
