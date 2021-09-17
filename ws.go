package main

import (
	"context"
	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/runner"
	"log"
	"sync"
	"time"

	"github.com/go-numb/go-ftx/realtime"
	"github.com/shopspring/decimal"
)

const (
	WS_RETRY_SLEEP_SECONDS = 3
)

type wsHandler struct {
	logger        *log.Logger
	ctx           context.Context
	wsRespCh      chan realtime.Response // response from exchange
	wsStopCh      chan bool              // graceful shutdown signal
	signalDoneCh  chan bool
	db            *db.DB
	runnerHandler *runnerHandler
}

func newWsHandler(l *log.Logger) *wsHandler {
	// TODO context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return &wsHandler{
		ctx:      ctx,
		wsRespCh: make(chan realtime.Response),
		wsStopCh: make(chan bool),
		logger:   l,
	}
}

func (h *wsHandler) setSignalDoneCh(ch chan bool) {
	h.signalDoneCh = ch
}

func (h *wsHandler) setDB(db *db.DB) {
	h.db = db
}

func (h *wsHandler) setRunnerHandler(srh *runnerHandler) {
	h.runnerHandler = srh
}

func (h *wsHandler) connect() {
	symbols, count, err := h.db.GetEnabledSymbols()
	if err != nil {
		h.logger.Fatal("err:", err)
	}
	if count == 0 {
		h.logger.Fatal("There is no symbol")
	}
	var pairs []string
	for _, s := range symbols {
		pairs = append(pairs, s.Name)
	}

	for {
		h.logger.Println("[ws] connecting...")
		err := realtime.Connect(h.ctx, h.wsRespCh, []string{"trades"}, pairs, h.logger)
		if err != nil {
			h.logger.Println("[ws] connection err:", err)
			h.logger.Printf("[ws] retry after %d seconds\n", WS_RETRY_SLEEP_SECONDS)
			time.Sleep(time.Second * WS_RETRY_SLEEP_SECONDS)
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

	// Filter incoming response by last timestamp
	var tradeLastTs sync.Map
	var tickerLastTs sync.Map

	for halted := false; !halted; {
		select {
		case v := <-h.wsRespCh:
			switch v.Type {
			// TODO keep??
			case realtime.TICKER:
				if h.ignoreResp(&tickerLastTs, v.Symbol) {
					break
				}

				if lastTickerAsk != v.Ticker.Ask || lastTickerBid != v.Ticker.Bid {
					lastTickerAsk = v.Ticker.Ask
					lastTickerBid = v.Ticker.Bid

					h.logger.Printf("%s  %.4f (%.2f)  %.4f (%.2f) %s\n", v.Symbol, v.Ticker.Ask, v.Ticker.AskSize, v.Ticker.Bid, v.Ticker.BidSize, v.Ticker.Time.Time.Format("2006-01-02 15:04:05"))
				}

			case realtime.TRADES:
				if h.ignoreResp(&tradeLastTs, v.Symbol) {
					break
				}

				// Get the last one as mark price
				trade := v.Trades[len(v.Trades)-1]
				mark := runner.Mark{
					Price: decimal.NewFromFloat(trade.Price),
					Time:  trade.Time,
				}
				// h.logger.Printf("%s  %4s | %.4f  %3.1f  %s\n", v.Symbol, trade.Side, trade.Price, trade.Size, trade.Time.Format("2006-01-02 15:04:05"))
				h.runnerHandler.broadcastMark(v.Symbol, mark)

			case realtime.ERROR:
				err = v.Results
				halted = true
			}

		case <-h.wsStopCh:
			// Graceful shutdown signal
			// Stop all strategy and sync.waitGroup all onging orders
			h.runnerHandler.stopAll()
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

func (h *wsHandler) ignoreResp(m *sync.Map, symbol string) bool {
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
