package main

import (
	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/exchange"
	"log"
	"time"

	"github.com/spf13/viper"
)

const (
	WS_RETRY_SLEEP_SECONDS = 3
)

type wsHandler struct {
	logger        *log.Logger
	wsStopCh      chan bool // graceful shutdown signal
	signalDoneCh  chan bool
	db            *db.DB
	runnerHandler *runnerHandler
}

func newWsHandler(l *log.Logger) *wsHandler {
	return &wsHandler{
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

func (h *wsHandler) setRunnerHandler(rh *runnerHandler) {
	h.runnerHandler = rh
}

func (h *wsHandler) connect() {
	rows, count, err := h.db.GetEnabledSymbols()
	if err != nil {
		h.logger.Fatal("err:", err)
	}
	if count == 0 {
		h.logger.Fatal("There is no symbol")
	}
	var symbols []string
	for _, symbol := range rows {
		symbols = append(symbols, symbol.Name)
	}

	// New exchange for ws
	ws, err := exchange.NewWsExchange(viper.GetString("DEFAULT_EXCHANGE"))
	if err != nil {
		h.logger.Fatal(err)
	}
	ws.SetBroadcastMarkFunc(h.runnerHandler.broadcastMark)
	ws.SetStopCh(h.wsStopCh)
	ws.SetStopAllFunc(h.runnerHandler.stopAll)
	debug := true

	// TODO FIXME if network is cut off, it won't come back
	for {
		h.logger.Println("[ws] connecting...")
		end, err := ws.ListenPublicTradesChannel(symbols, debug)
		if err != nil {
			h.logger.Printf("[ws] error: %s", err)
		}
		if end {
			break
		}
		h.logger.Println("[ws] disconnected")
		h.logger.Printf("[ws] retry after %d seconds\n", WS_RETRY_SLEEP_SECONDS)
		time.Sleep(time.Second * WS_RETRY_SLEEP_SECONDS)
	}
	h.signalDoneCh <- true
}

func (h *wsHandler) close() {
	h.logger.Println("[ws] closing...")
	close(h.wsStopCh)
}
