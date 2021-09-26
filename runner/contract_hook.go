package runner

import (
	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/exchange"
	"crypto-trading-bot-main/message"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"
	"crypto-trading-bot-main/strategy/trigger"
	"errors"
	"fmt"
	"log"
	"strings"
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

// Check exchange_orders_details, halt the strategy if the data is out of sync
func ValidateExchangeOrdersDetails(cs *db.ContractStrategy) error {
	switch contract.Status(cs.PositionStatus) {
	case contract.CLOSED:
		if len(cs.ExchangeOrdersDetails) > 0 {
			return errors.New("position status: 'CLOSED', 'exchange_orders_details' isn't empty")
		}
	case contract.OPENED:
		if len(cs.ExchangeOrdersDetails) == 0 {
			return errors.New("position status: 'OPENED', 'exchange_orders_details' is empty")
		}
		entryOrder, ok := cs.ExchangeOrdersDetails["entry_order"].(map[string]interface{})
		if !ok {
			return errors.New("position status: 'OPENED', 'exchange_orders_details.entry_order' is missing")
		}
		_, ok = entryOrder["size"].(string)
		if !ok {
			return errors.New("position status: 'OPENED', 'exchange_orders_details.entry_order.size' is missing")
		}
	case contract.UNKNOWN:
		return errors.New("unknown status")
	}
	return nil
}

func (ch *contractHook) EntryTriggered(c *contract.Contract, t time.Time, p decimal.Decimal) (decimal.Decimal, bool, error) {
	if err := ValidateExchangeOrdersDetails(ch.contractStrategy); err != nil {
		return p, true, err
	}

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
		text := fmt.Sprintf("[Warn] %s strategies can only be triggered once at a time. '%s %s $%s' will be ignored and disabled", ch.contractStrategy.Symbol, order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, ch.contractStrategy.Margin)
		ch.notify(text)
		return p, true, nil
	}

	// Calculate the size
	size := ch.contractStrategy.Margin.Div(p)

	// Place entry order
	orderId, err := ch.exchange.PlaceEntryOrder(ch.contractStrategy.Symbol, order.Side(ch.contractStrategy.Side), size)
	if err != nil {
		text := fmt.Sprintf("[Error] Failed to place entry order, err: %v", err)
		ch.notify(text)
		return p, false, fmt.Errorf("EntryTriggered - failed to place entry order, err: %v", err)
	}

	// Check position - retyr 30 times, interval 2 secs
	orderInfo, count, err := ch.exchange.RetryGetPosition(orderId, 30, 2)
	if err != nil {
		text := fmt.Sprintf("[Error] Failed to get open position, err: %v", err)
		ch.notify(text)
		return p, true, fmt.Errorf("EntryTriggered - failed to get open position, err: %v", err)
	}
	if count == 0 {
		text := fmt.Sprint("[Warn] Entry order has been placed, but can't find any open position. please check and reset your position and order")
		ch.notify(text)
		return p, true, fmt.Errorf("EntryTriggered - entry order has been placed, but can't find any open position")
	}

	// Notification
	text = fmt.Sprintf("[Entry] '%s %s' has been triggered @%s (margin: $%s, fee: $%.1f)", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, orderInfo["price"].(string), ch.contractStrategy.Margin.StringFixed(0), orderInfo["fee"].(float64))
	ch.notify(text)

	// Update data for orders info
	exchangeOrdersDetails := datatypes.JSONMap{
		"entry_order": orderInfo,
	}

	// For memory data
	ch.contractStrategy.PositionStatus = int64(contract.OPENED)
	ch.contractStrategy.ExchangeOrdersDetails = exchangeOrdersDetails
	ch.contractStrategy.LastPositionAt = orderInfo["time"].(time.Time)

	// For DB
	contractStrategy := map[string]interface{}{
		"position_status":         int64(contract.OPENED),
		"exchange_orders_details": exchangeOrdersDetails,
		"last_position_at":        ch.contractStrategy.LastPositionAt,
	}
	_, err = ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy)
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return p, true, fmt.Errorf("EntryTriggered - failed to update 'exchange_orders_details', err: %v", err)
	}

	entryPrice, err := decimal.NewFromString(orderInfo["price"].(string))
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return p, true, fmt.Errorf("EntryTriggered - failed to convert 'price' from order info, err: %v", err)
	}

	return entryPrice, false, nil
}

func (ch *contractHook) StopLossTriggerCreated(c *contract.Contract) (bool, error) {
	var text string

	// entry_type 'limit' and 'baseline' both are using Limit Trigger, time doesn't matter
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

	orderInfo, err := ch.closeOpenPosition("StopLossTriggered")
	if err != nil {
		// The error indicates that the position has been closed already
		if !strings.Contains(err.Error(), "Invalid reduce-only order") {
			return true, fmt.Errorf("StopLossTriggered - failed to close position, err: %v", err)
		}
		// NOTE position has been closed by FTX
		text := fmt.Sprintf("[Stop-loss] '%s %s' position has been closed by stop-loss trigger order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
	}

	// Cancel trigger if not triggered yet
	var stopLossOrderId float64
	orderInfo, ok := ch.contractStrategy.ExchangeOrdersDetails["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrderId, ok = orderInfo["order_id"].(float64)
		if !ok {
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
			ch.notify(text)
			return true, fmt.Errorf("StopLossTriggered - stop-loss 'order_id' is missing")
		}
		if err = ch.exchange.RetryCancelOpenTriggerOrder(int64(stopLossOrderId), 20, 2); err != nil {
			if !strings.Contains(err.Error(), "Order already closed") {
				text = fmt.Sprintf("[Error] Failed to cancel %s stop-loss order, err: %v", ch.contractStrategy.Symbol, err)
				ch.notify(text)
				return true, err
			}
			// NOTE stop-loss trigger order has been closed by FTX
		}
	} else {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("StopLossTriggered - stop-loss 'stop_loss_order' is missing")
	}

	// Reset status and exchange_orders_details
	contractStrategy := map[string]interface{}{
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	_, err = ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy)
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("StopLossTriggered - failed to update 'position_status', err: %v", err)
	}
	ch.contractStrategy.PositionStatus = int64(contract.CLOSED)
	return false, nil
}

func (ch *contractHook) EntryBaselineTriggerUpdated(c *contract.Contract) {
	text := fmt.Sprintf("[Info] '%s %s' entry baseline has been updated", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)

	// Send new baseline
	t := c.EntryOrder.(*order.Entry).BaselineTrigger
	// trigger shouldn't be 'nil', but just in case that it won't blow up
	if t != nil {
		p1 := t.(*trigger.Line).Price1
		t1 := t.(*trigger.Line).Time1
		p2 := t.(*trigger.Line).Price2
		t2 := t.(*trigger.Line).Time2
		text = fmt.Sprintf("[Info] New entry baseline:\nPoint 1: $%s, '%s'\nPoint 2: $%s, '%s'", p1, t1.Format("2006-01-02 15:04"), p2, t2.Format("2006-01-02 15:04"))
		ch.notify(text)
	}
}

func (ch *contractHook) TakeProfitTriggered(c *contract.Contract) error {
	text := fmt.Sprintf("[Take-profit] '%s %s' has been triggered", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
	ch.notify(text)
	err := ch.closePosition()

	// Update memory data
	ch.contractStrategy.Enabled = 0

	// NOTE Update DB data by event channel
	// Let the caller to decide whether it should be reset by returning `halted` and `err`
	return err
}

// NOTE datatypes.JSONMap will escapte `<` into `\u003c`, but it's fine. It can still be unmarchal and turned back to `=` without issue
// NOTE datatypes.JSONMap will turm time into `2021-09-15T04:00:00Z`
func (ch *contractHook) ParamsUpdated(c *contract.Contract) (bool, error) {
	params := datatypes.JSONMap{
		"entry_type":  c.EntryType,
		"entry_order": c.EntryOrder,
	}
	if c.StopLossOrder != nil {
		params["stop_loss_order"] = c.StopLossOrder
	}
	if c.TakeProfitOrder != nil {
		params["take_profit_order"] = c.TakeProfitOrder
	}

	// Update db
	contractStrategy := map[string]interface{}{
		"params": params,
	}
	if _, err := ch.db.UpdateContractStrategy(ch.contractStrategy.Uuid, contractStrategy); err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return true, fmt.Errorf("ParamsUpdated - failed to update 'params', err: %v", err)
	}

	return false, nil
}

// Let the caller to decide whether it should be reset by returning `halted` and `err`
func (ch *contractHook) closePosition() error {
	var text string

	orderInfo, err := ch.closeOpenPosition("closePosition")
	if err != nil {
		return fmt.Errorf("closePosition > %v", err)
	}

	// Cancel stop-loss order if exists
	// It's possible that there is no order_id for stop-loss order that will happen when the position has been opened but something went wrong before stop-loss order is created
	var stopLossOrderId int64
	orderInfo, ok := ch.contractStrategy.ExchangeOrdersDetails["stop_loss_order"].(map[string]interface{})
	if ok {
		stopLossOrderId = int64(orderInfo["order_id"].(float64))
		err = ch.exchange.RetryCancelOpenTriggerOrder(stopLossOrderId, 20, 2)
		if err != nil {
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

func (ch *contractHook) closeOpenPosition(caller string) (map[string]interface{}, error) {
	var text string

	// Close position
	size, err := decimal.NewFromString(ch.contractStrategy.ExchangeOrdersDetails["entry_order"].(map[string]interface{})["size"].(string))
	if err != nil {
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol)
		ch.notify(text)
		return map[string]interface{}{}, fmt.Errorf("closeOpenPosition - failed to convert 'size' from order info, err: %v", err)
	}

	orderId, err := ch.exchange.RetryClosePosition(ch.contractStrategy.Symbol, order.Side(ch.contractStrategy.Side), size, 30, 2)
	if err != nil {
		// position could be closed by stop-loss trigger order, it's fine for caller `StopLossTriggered`
		if caller == "closePosition" {
			text = fmt.Sprintf("[Error] %s %s - failed to close position, please check and reset your position and order, err: %v", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, err)
			ch.notify(text)
		}
		return map[string]interface{}{}, fmt.Errorf("closeOpenPosition - failed to close position, err: %v", err)
	}

	// Check position
	orderInfo, count, err := ch.exchange.RetryGetPosition(orderId, 30, 2)
	if err != nil {
		text := fmt.Sprintf("[Error] Failed to get position from the position just opened, err: %v", err)
		ch.notify(text)
		return map[string]interface{}{}, fmt.Errorf("closeOpenPosition - failed to get position, err: %v", err)
	}
	if count == 0 {
		text := fmt.Sprint("[Warn] Not sure whether the position has been closed. please check and reset your position and order")
		ch.notify(text)
		return map[string]interface{}{}, errors.New("closeOpenPosition - no position can be found")
	}

	// Notification
	text = fmt.Sprintf("[Info] '%s %s' position has been closed @%s (fee: $%.1f)", order.TranslateSideByInt(ch.contractStrategy.Side), ch.contractStrategy.Symbol, orderInfo["price"].(string), orderInfo["fee"].(float64))
	ch.notify(text)

	return orderInfo, nil
}

func (ch *contractHook) notify(text string) {
	ch.logger.Printf("sid: %s uid: %s sym: %s text: '%s'", ch.contractStrategy.Uuid, ch.contractStrategy.UserUuid, ch.contractStrategy.Symbol, text)
	go ch.sender.Send(ch.user.TelegramChatId, text)
}
