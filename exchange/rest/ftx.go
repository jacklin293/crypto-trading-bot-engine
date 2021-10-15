package rest

import (
	"crypto-trading-bot-engine/strategy/order"
	"errors"
	"fmt"
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
	subaccount := data["subaccount"].(string)

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
	_, ok = data["subaccount"].(string)
	if !ok {
		return errors.New("'subaccount' is missing")
	}
	return nil
}

func (rest *FtxRest) GetAccountInfo() (map[string]interface{}, error) {
	r := make(map[string]interface{})
	info, err := rest.client.Account.GetAccountInformation()
	if err != nil {
		return r, err
	}

	r["collateral"] = info.Collateral          // decimal.Decimal
	r["free_collateral"] = info.FreeCollateral // decimal.Decimal
	r["maker_fee"] = info.MakerFee             // decimal.Decimal
	r["taker_fee"] = info.TakerFee             // decimal.Decimal
	r["username"] = info.Username              // string
	r["leverage"] = info.Leverage              // decimal.Decimal
	return r, nil
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
	return order.ID, err
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

// &{
//		Cost:78.66075
//		EntryPrice:52.4405
//		EstimatedLiquidationPrice:0
//		Future:FTT-PERP				// string, other fields are all decimal.Decimal
//		InitialMarginRequirement:0.2
//		LongOrderSize:0
//		MaintenanceMarginRequirement:0.03
//		NetSize:1.5
//		OpenSize:1.5
//		RealizedPnl:-4.87398562
//		ShortOrderSize:0
//		Side:buy					// string
//		Size:1.5
//		UnrealizedPnl:0
//		CollateralUsed:15.73215
// }
func (rest *FtxRest) GetPosition(symbol string) (map[string]interface{}, error) {
	p := make(map[string]interface{})
	positions, err := rest.client.Account.GetPositions()
	if err != nil {
		return p, err
	}

	for _, position := range positions {
		if position.Future == symbol {
			// NOTE cost and entry_price are useless, they are dynamic instead of the data happened
			p["cost"] = position.Cost.Abs().String() // make number positive no matter it's long or short
			p["entry_price"] = position.EntryPrice.String()
			p["size"] = position.Size.String()
			p["symbol"] = position.Future
			switch models.Side(position.Side) {
			case models.Buy:
				p["side"] = float64(order.LONG)
			case models.Sell:
				p["side"] = float64(order.SHORT)
			default:
				return p, fmt.Errorf("side '%s' not supported", position.Side)
			}
			return p, err
		}
	}
	return p, fmt.Errorf("failed to get %s position", symbol)
}

func (rest *FtxRest) RetryGetPosition(symbol string, retry int64, interval int64) (p map[string]interface{}, err error) {
	for i := int64(0); i <= retry; i++ {
		p, err = rest.GetPosition(symbol)
		if err != nil {
			if strings.Contains(err.Error(), "wrong side") {
				break
			}
			log.Println("RetryGetPosition err:", err)
			time.Sleep(time.Second * time.Duration(interval))
			continue
		}
		break
	}
	return
}

func (rest *FtxRest) ClosePosition(symbol string, side order.Side, size decimal.Decimal) error {
	reduceOnly := true
	_, err := rest.client.Orders.PlaceOrder(&models.PlaceOrderPayload{
		Market:     symbol,
		Side:       rest.translateAndFlipSide(side),
		Type:       models.MarketOrder,
		Size:       size,
		ReduceOnly: &reduceOnly,
	})
	return err
}

// NOTE Don't use this, because it would loop until end when the position has been closed. The error will always get 'Status Code: 400    Error: Invalid reduce-only order'
// func (rest *FtxRest) RetryClosePosition(symbol string, side order.Side, size decimal.Decimal, retry int64, interval int64) (err error) {
//	for i := int64(0); i <= retry; i++ {
//		if err = rest.ClosePosition(symbol, side, size); err != nil {
//			if rest.ignoreError(err) {
//				break
//			}
//			log.Printf("RetryClosePosition err: %v", err)
//			time.Sleep(time.Second * time.Duration(interval))
//			continue
//		}
//		break
//	}
//	return
// }

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
	// NOTE: update: found that it could be thrown for other unknow reason
	// if strings.Contains(err.Error(), "Invalid reduce-only order") {
	//	return true
	// }

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

// NOTE This function can't be used to check position status, it only returns the same data when it is created
//      FTX API won't return completed filled data
// func (rest *FtxRest) GetFill(orderId int64) (r map[string]interface{}, err error) {
//	r = make(map[string]interface{})
//	// NOTE This endpoint only returns the data that the last order gets filled, which means that it can't be used for
//	//      getting complete filled size
//	fills, err := rest.client.Fills.GetFills(&models.GetFillsParams{
//		OrderID: &orderId,
//	})
//	if err != nil {
//		return
//	}
//	for _, f := range fills {
//		if f.OrderID == orderId {
//			// NOTE the reason why to convert type is for making them more consistent when they are retrieved from DB
//			//      , as int64 will be turned into float64 after it was unmarshelled
//			r["fee_rate"] = f.FeeRate          // float64
//			r["order_id"] = float64(f.OrderID) // original: int64
//			r["price"] = f.Price.String()      // original: decimal.Decimal
//			r["time"] = f.Time.Time
//
//			// NOTE They could be from the last filled order, so the data can't be trusted
//			// r["fee"] = f.Fee                   // float64
//			// r["size"] = f.Size.String()        // original: decimal.Decimal
//			return
//		}
//	}
//	return
// }
//
// func (rest *FtxRest) RetryGetFill(orderId int64, retry int64, interval int64) (r map[string]interface{}, err error) {
//	for i := int64(0); i <= retry; i++ {
//		r, err = rest.GetFill(orderId)
//		if err != nil {
//			log.Println("RetryGetFill err:", err)
//			time.Sleep(time.Second * time.Duration(interval))
//			continue
//		}
//		if len(r) == 0 {
//			log.Println("RetryGetFill order_id not matched")
//			time.Sleep(time.Second * time.Duration(interval))
//			continue
//		}
//		// success
//		break
//	}
//	return
// }
