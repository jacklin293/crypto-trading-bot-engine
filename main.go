package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-numb/go-ftx/realtime"
)

type handler struct {
	ctx      context.Context
	wsRespCh chan realtime.Response
	wsErrCh  chan error
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wsRespCh := make(chan realtime.Response)
	h := handler{
		ctx:      ctx,
		wsRespCh: wsRespCh,
	}
	go h.handleWs()

	h.listen()
}

func (h *handler) handleWs() {
	// TODO ftx websocket/Users/jacklin/go/pkg/mod/github.com/go-numb/go-ftx@v0.0.0-20210814041904-1aef7a7101bb/realtime/websocket.go:161: [ERROR]: msg error: read tcp 10.5.0.2:60885->104.18.26.153:443: read: connection reset by peer
	// TODO handle error returned from Connect
	// TODO for

	h.connectWs()
	for {
		select {
		// TODO Confirm if it works
		case err := <-h.wsErrCh:
			fmt.Println("capture err:", err)
			// reconnect ws when error happens
			h.connectWs()
		}
	}
}
func (h *handler) connectWs() {
	pairs := getPairs()
	err := realtime.Connect(h.ctx, h.wsRespCh, []string{"trades"}, pairs, nil)
	h.wsErrCh <- err
}

func (h *handler) listen() {
	var lastTickerAsk, lastTickerBid float64
	var now, lastTs int64
	for {
		select {
		case v := <-h.wsRespCh:
			// TODO by pair and by type
			now = time.Now().Unix()
			if now == lastTs {
				continue
			}

			switch v.Type {
			case realtime.TICKER:
				if lastTickerAsk != v.Ticker.Ask || lastTickerBid != v.Ticker.Bid {
					lastTickerAsk = v.Ticker.Ask
					lastTickerBid = v.Ticker.Bid

					fmt.Printf("%s  %.4f (%.2f)  %.4f (%.2f) %s\n", v.Symbol, v.Ticker.Ask, v.Ticker.AskSize, v.Ticker.Bid, v.Ticker.BidSize, v.Ticker.Time.Time.Format("2006-01-02 15:04:05"))
				}

			case realtime.TRADES: // TODO can be used as mark price
				trade := v.Trades[len(v.Trades)-1]
				fmt.Printf("%s  %4s | %.4f  %3.1f  %s\n", v.Symbol, trade.Side, trade.Price, trade.Size, trade.Time.Format("2006-01-02 15:04:05"))
			case realtime.UNDEFINED:
				fmt.Printf("UNDEFINED %s	%s\n", v.Symbol, v.Results.Error())
			}

			lastTs = time.Now().Unix()
		}
	}
}

// TODO Get pairs from redis, if not exists, read from DB
func getPairs() []string {
	return []string{"BTC-PERP"}
}
