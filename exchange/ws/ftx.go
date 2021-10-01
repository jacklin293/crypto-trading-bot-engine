package ws

import (
	"context"
	"crypto-trading-bot-engine/strategy/contract"
	"fmt"
	"sync"
	"time"

	"github.com/grishinsana/goftx"
)

type FtxWs struct {
	broadcastMark func(string, contract.Mark)
	stopCh        chan bool
	stopAll       func()
}

func NewFtxWs() *FtxWs {
	return &FtxWs{}
}

func (ws *FtxWs) SetBroadcastMarkFunc(f func(string, contract.Mark)) {
	ws.broadcastMark = f
}

func (ws *FtxWs) SetStopCh(ch chan bool) {
	ws.stopCh = ch
}

func (ws *FtxWs) SetStopAllFunc(f func()) {
	ws.stopAll = f
}

func (ws *FtxWs) ListenPublicTradesChannel(symbols []string, debug bool) (end bool, err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := goftx.New()
	client.Stream.SetDebugMode(debug)

	trades, err := client.Stream.SubscribeToTrades(ctx, symbols...)
	if err != nil {
		return end, err
	}

	// Filter incoming response by last timestamp
	var lastTradeTs sync.Map

	for halted := false; halted != true; {
		select {
		case <-ctx.Done():
			halted = true
			break
		case trade, ok := <-trades:
			if !ok {
				err = fmt.Errorf("unknown error from ws, resp: %+v", trade)
				halted = true
				break
			}
			if ws.ignoreResp(&lastTradeTs, trade.BaseResponse.Symbol) {
				break
			}

			mark := contract.Mark{
				Price: trade.Trade.Price,
				Time:  trade.Trade.Time,
			}
			ws.broadcastMark(trade.BaseResponse.Symbol, mark)
		case <-ws.stopCh:
			ws.stopAll()
			end = true
			halted = true
			break
		}
	}

	return end, err
}

func (ws *FtxWs) ignoreResp(m *sync.Map, symbol string) bool {
	ts := time.Now().UnixMilli()
	lastTs, ok := m.Load(symbol)
	if ok {
		// the rate of checking prices during a second e.g. '< 200' means 4 times per second
		if ts-lastTs.(int64) < 200 {
			return true
		}
	}
	m.Store(symbol, ts)
	return false
}
