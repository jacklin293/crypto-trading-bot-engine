package exchange

import (
	"crypto-trading-bot-main/exchange/rest"
	"crypto-trading-bot-main/exchange/ws"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"
	"fmt"

	"github.com/shopspring/decimal"
)

type Exchanger interface {
	NewClient(map[string]interface{}) error
	PlaceEntryOrder(string, order.Side, decimal.Decimal) (int64, error)
	PlaceStopLossOrder(string, order.Side, decimal.Decimal, decimal.Decimal) (int64, error)
	RetryPlaceStopLossOrder(string, order.Side, decimal.Decimal, decimal.Decimal, int64, int64) (int64, error)
	GetPosition(int64) (map[string]interface{}, int64, error)
	RetryGetPosition(int64, int64, int64) (map[string]interface{}, int64, error)
	ClosePosition(string, order.Side, decimal.Decimal) (int64, error)
	RetryClosePosition(string, order.Side, decimal.Decimal, int64, int64) (int64, error)
	CancelOpenTriggerOrder(int64) error
	RetryCancelOpenTriggerOrder(int64, int64, int64) error
}

type WsExchanger interface {
	SetBroadcastMarkFunc(func(string, contract.Mark))
	SetStopCh(chan bool)
	SetStopAllFunc(func())
	ListenPublicTradesChannel([]string, bool) (bool, error)
}

func NewExchange(name string, data map[string]interface{}) (ex Exchanger, err error) {
	switch name {
	case "ftx":
		ex = rest.NewFtxRest()
		err = ex.NewClient(data)
	default:
		err = fmt.Errorf("exchange '%s' no supported", name)
	}
	return
}

func NewWsExchange(name string) (ex WsExchanger, err error) {
	switch name {
	case "ftx":
		ex = ws.NewFtxWs()
	default:
		err = fmt.Errorf("exchange '%s' no supported", name)
	}
	return
}
