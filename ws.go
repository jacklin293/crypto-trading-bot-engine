package main

import (
	"context"
	"log"
	"time"

	"github.com/go-numb/go-ftx/realtime"
)

const (
	WS_RETRY_SECONDS = 3
)

type wsHandler struct {
	logger       *log.Logger
	ctx          context.Context
	wsRespCh     chan realtime.Response // response from exchange
	wsStopCh     chan bool              // graceful shutdown signal
	signalDoneCh chan bool
}

func newWsHandler(l *log.Logger, ch chan bool) *wsHandler {
	// FIXME context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return &wsHandler{
		ctx:          ctx,
		wsRespCh:     make(chan realtime.Response),
		wsStopCh:     make(chan bool),
		logger:       l,
		signalDoneCh: ch,
	}
}

func (h *wsHandler) connect() {
	pairs := getPairs()
	for {
		h.logger.Println("[ws] connecting...")
		err := realtime.Connect(h.ctx, h.wsRespCh, []string{"trades"}, pairs, h.logger)
		if err != nil {
			h.logger.Println("[ws] connection err:", err)
			h.logger.Printf("[ws] retry after %d seconds\n", WS_RETRY_SECONDS)
			time.Sleep(time.Second * WS_RETRY_SECONDS)
			continue
		}
		h.logger.Println("[ws] listening...")
		err = h.listen()
		// if it's connection issue, just reconnect
		if err == nil {
			break
		}
	}
	h.signalDoneCh <- true
}

func (h *wsHandler) listen() (err error) {
	var lastTickerAsk, lastTickerBid float64
	var now, lastTs int64

	for halted := false; !halted; {
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

					h.logger.Printf("%s  %.4f (%.2f)  %.4f (%.2f) %s\n", v.Symbol, v.Ticker.Ask, v.Ticker.AskSize, v.Ticker.Bid, v.Ticker.BidSize, v.Ticker.Time.Time.Format("2006-01-02 15:04:05"))
				}
			case realtime.TRADES: // TODO can be used as mark price
				trade := v.Trades[len(v.Trades)-1]
				h.logger.Printf("%s  %4s | %.4f  %3.1f  %s\n", v.Symbol, trade.Side, trade.Price, trade.Size, trade.Time.Format("2006-01-02 15:04:05"))
			case realtime.ERROR:
				err = v.Results
				halted = true
			}
			lastTs = time.Now().Unix()
		case <-h.wsStopCh:
			// Graceful shutdown signal
			halted = true
		}
		if halted {
			break
		}
	}
	return
}

func (h *wsHandler) close() {
	h.logger.Println("[ws] closing...")
	close(h.wsStopCh)
}
