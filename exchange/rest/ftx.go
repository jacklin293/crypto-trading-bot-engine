package rest

import (
	"crypto-trading-bot-engine/strategy/order"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/grishinsana/goftx"
	"github.com/grishinsana/goftx/models"
	"github.com/shopspring/decimal"
)

type FtxRest struct {
	client *goftx.Client
}

func NewFtxRest() *FtxRest {
	return &FtxRest{}
}

func (rest *FtxRest) NewClient(data map[string]interface{}) (err error) {
	err = rest.validate(data)
	if err != nil {
		return
	}
	key := data["api_key"].(string)
	secret := data["api_secret"].(string)
	subaccount := data["subaccount_name"].(string)

	rest.client = goftx.New(
		goftx.WithAuth(key, secret, subaccount),
		goftx.WithHTTPClient(&http.Client{
			Timeout: 5 * time.Second,
		}),
	)
	return
}

func (rest *FtxRest) validate(data map[string]interface{}) error {
	_, ok := data["api_key"].(string)
	if !ok {
		return errors.New("'api_key' is missing")
	}
	_, ok = data["api_secret"].(string)
	if !ok {
		return errors.New("'api_secret' is missing")
	}
	_, ok = data["subaccount_name"].(string)
	if !ok {
		return errors.New("'subaccount_name' is missing")
	}
	return nil
}

func (rest *FtxRest) PlaceEntryOrder(symbol string, side order.Side, size decimal.Decimal) (int64, error) {
	order, err := rest.client.Orders.PlaceOrder(&models.PlaceOrderPayload{
		Market: symbol,
		Side:   rest.translateSide(side),
		Type:   models.MarketOrder,
		Size:   size,
	})
	if err != nil {
		return 0, err
	}
	return order.ID, nil
}

func (rest *FtxRest) PlaceStopLossOrder(symbol string, side order.Side, price decimal.Decimal, size decimal.Decimal) (int64, error) {
	reduceOnly := true
	retryUntilFilled := true
	triggerPrice := price
	order, err := rest.client.Orders.PlaceTriggerOrder(&models.PlaceTriggerOrderPayload{
		Market:           symbol,
		Side:             rest.translateAndFlipSide(side),
		Size:             size,
		Type:             models.Stop,
		ReduceOnly:       &reduceOnly,
		RetryUntilFilled: &retryUntilFilled,
		TriggerPrice:     &triggerPrice,
	})
	if err != nil {
		return 0, err
	}
	return order.ID, nil
}

// retry: retry times
// interval: sleep seconds
func (rest *FtxRest) RetryPlaceStopLossOrder(symbol string, side order.Side, price decimal.Decimal, size decimal.Decimal, retry int64, interval int64) (orderId int64, err error) {
	for i := int64(0); i <= retry; i++ {
		orderId, err = rest.PlaceStopLossOrder(symbol, side, price, size)
		if err != nil {
			if rest.ignoreError(err) {
				break
			}
			log.Println("RetryPlaceStopLossOrder err:", err)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		break
	}
	return
}

// NOTE There is no way to check if position has been closed or not
//      This function can't be used to check position status, it only returns the same data when it is created
func (rest *FtxRest) GetPosition(orderId int64) (r map[string]interface{}, count int64, err error) {
	r = make(map[string]interface{})
	fills, err := rest.client.Fills.GetFills(&models.GetFillsParams{
		OrderID: &orderId,
	})
	count = int64(len(fills))
	if err != nil {
		return
	}
	for _, f := range fills {
		if f.OrderID == orderId {
			// NOTE the reason why to convert type is for making them more consistent when they are retrieved from DB
			//      , as int64 will be turned into float64 after it was unmarshelled
			r["fee"] = f.Fee                   // float64
			r["fee_rate"] = f.FeeRate          // float64
			r["order_id"] = float64(f.OrderID) // original: int64
			r["price"] = f.Price.String()      // original: decimal.Decimal
			r["size"] = f.Size.String()        // original: decimal.Decimal
			r["time"] = f.Time.Time
			return
		}
	}
	return
}

// retry: retry times
// interval: sleep seconds
func (rest *FtxRest) RetryGetPosition(orderId int64, retry int64, interval int64) (r map[string]interface{}, count int64, err error) {
	for i := int64(0); i <= retry; i++ {
		r, count, err = rest.GetPosition(orderId)
		if err != nil {
			if rest.ignoreError(err) {
				break
			}
			log.Println("RetryGetPosition err:", err)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		if count == 0 {
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		// success
		break
	}
	return
}

func (rest *FtxRest) ClosePosition(symbol string, side order.Side, size decimal.Decimal) (int64, error) {
	reduceOnly := true
	order, err := rest.client.Orders.PlaceOrder(&models.PlaceOrderPayload{
		Market:     symbol,
		Side:       rest.translateAndFlipSide(side),
		Type:       models.MarketOrder,
		Size:       size,
		ReduceOnly: &reduceOnly,
	})
	if err != nil {
		return 0, err
	}
	return order.ID, nil
}

func (rest *FtxRest) RetryClosePosition(symbol string, side order.Side, size decimal.Decimal, retry int64, interval int64) (orderId int64, err error) {
	for i := int64(0); i <= retry; i++ {
		orderId, err = rest.ClosePosition(symbol, side, size)
		if err != nil {
			if rest.ignoreError(err) {
				break
			}
			log.Println("RetryClosePosition err:", err)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		break
	}
	return
}

func (rest *FtxRest) CancelOpenTriggerOrder(orderId int64) error {
	return rest.client.Orders.CancelOpenTriggerOrder(orderId)
}

func (rest *FtxRest) RetryCancelOpenTriggerOrder(orderId int64, retry int64, interval int64) (err error) {
	for i := int64(0); i <= retry; i++ {
		err = rest.client.Orders.CancelOpenTriggerOrder(orderId)
		if err != nil {
			if rest.ignoreError(err) {
				break
			}
			log.Println("RetryCancelOpenTriggerOrder err:", err)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		break
	}
	return
}

func (rest *FtxRest) translateSide(s order.Side) models.Side {
	switch s {
	case order.LONG:
		return models.Buy
	case order.SHORT:
		return models.Sell
	}
	return ""
}

func (rest *FtxRest) translateAndFlipSide(s order.Side) models.Side {
	switch s {
	case order.LONG:
		return models.Sell
	case order.SHORT:
		return models.Buy
	}
	return ""
}

func (rest *FtxRest) ignoreError(err error) bool {
	// Scenario: try to place an order with small size for a symbol
	// Reproduction: spend below $5 dollars
	if strings.Contains(err.Error(), "Size too small") {
		return true
	}

	// Scenario: try to get an open position that has been closed already
	// Reproduction: get entry_order triggered, then sleep 30s, close position manually on app during this 30s, then get position
	if strings.Contains(err.Error(), "Invalid reduce-only order") {
		return true
	}

	// Scenario: try to get an open trigger order that has been closed
	// Reproduction: get stop-loss order created, then sleep 30s, close order manually on app during this 30s, then get trigger order
	if strings.Contains(err.Error(), "Order already closed") {
		return true
	}

	// Scenario: try to place an order with insufficient margin
	// Reproduction: place an order with money more than you have
	if strings.Contains(err.Error(), "Account does not have enough margin for order") {
		return true
	}

	return false
}

/*
No need?
func (rest *FtxRest) OpenStopTriggerOrderExists(symbol string, orderId int64) (existed bool, err error) {
	t := models.Stop
	orders, err := rest.client.Orders.GetOpenTriggerOrders(&models.GetOpenTriggerOrdersParams{
		Market: &symbol,
		Type:   &t,
	})
	if err != nil {
		return false, nil
	}
	for _, order := range orders {
		if order.ID == orderId {
			return true, nil
		}
	}
	return false, nil
}

func (rest *FtxRest) CancelOrder(orderId int64) error {
	return rest.client.Orders.CancelOrder(orderId)
}

func (rest *FtxRest) RetryCancelOrder(orderId int64, retry int64, interval int64) (err error) {
	for i := int64(0); i <= retry; i++ {
		err = rest.client.Orders.CancelOrder(orderId)
		if err != nil {
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		break
	}
	return
}
*/
