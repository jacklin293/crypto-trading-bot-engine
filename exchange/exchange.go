package exchange

import (
	"bytes"
	"crypto-trading-bot-engine/exchange/rest"
	"crypto-trading-bot-engine/exchange/ws"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
	"crypto-trading-bot-engine/util/aes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/spf13/viper"
)

type Exchanger interface {
	NewClient(map[string]interface{}) error
	GetAccountInfo() (map[string]interface{}, error)
	PlaceEntryOrder(string, order.Side, decimal.Decimal) (int64, error)
	PlaceStopLossOrder(string, order.Side, decimal.Decimal, decimal.Decimal) (int64, error)
	RetryPlaceStopLossOrder(string, order.Side, decimal.Decimal, decimal.Decimal, int64, int64) (int64, error)
	CancelStopLossOrder(int64) error
	ClosePosition(string, order.Side, decimal.Decimal) error
	CancelOpenTriggerOrder(int64) error
	RetryCancelOpenTriggerOrder(int64, int64, int64) error
	GetPosition(string) (map[string]interface{}, error)
	RetryGetPosition(string, int64, int64) (map[string]interface{}, error)
	StopLostOrderExists(string, int64) (bool, error)
}

type WsExchanger interface {
	SetBroadcastMarkFunc(func(string, contract.Mark))
	SetStopCh(chan bool)
	SetStopAllFunc(func())
	ListenPublicTradesChannel([]string, bool) (bool, error)
}

// Private endpoints
func NewExchange(exName string, encryptedAesPair string) (ex Exchanger, err error) {
	switch exName {
	case "FTX":
		exData := make(map[string]interface{})
		exData, err = validateData(exName, encryptedAesPair)
		if err != nil {
			break
		}
		ex = rest.NewFtxRest()
		err = ex.NewClient(exData)
	default:
		err = fmt.Errorf("exchange '%s' no supported", exName)
	}
	return
}

// Public endpoints
func NewWsExchange(exName string) (ex WsExchanger, err error) {
	switch exName {
	case "FTX":
		ex = ws.NewFtxWs()
	default:
		err = fmt.Errorf("exchange '%s' no supported", exName)
	}
	return
}

func validateData(exName string, encryptedAesPair string) (exData map[string]interface{}, err error) {
	encryptedData := strings.Split(encryptedAesPair, ";")

	if len(encryptedData) < 2 {
		return exData, errors.New("invalid api key data")
	}

	iv64 := encryptedData[0]
	data64 := encryptedData[1]
	key, err := hex.DecodeString(viper.GetString("AES_PRIVATE_KEY"))
	if err != nil {
		return exData, err
	}
	text, err := aes.Decrypt(key, iv64, data64)
	if err != nil {
		return exData, err
	}
	// NOTE for fixing `invalid character '\x00' after top-level value`
	text = bytes.Trim(text, "\x00")

	// Exchange API Key
	exMap := make(map[string]interface{})
	if err = json.Unmarshal(text, &exMap); err != nil {
		return
	}

	// Make sure there is a key for specific exchange
	exData, ok := exMap[exName].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("key '%s' is missing in the map", exName)
		return
	}
	return exData, nil
}
