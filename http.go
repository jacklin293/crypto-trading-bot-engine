package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/spf13/viper"
)

const (
	HTTP_SERVER_SHUTDOWN_TIMEOUT = 30
)

type httpHandler struct {
	server        *http.Server
	logger        *log.Logger
	runnerHandler *runnerHandler
}

func newHttpHandler(l *log.Logger) *httpHandler {
	port := viper.GetString("HTTP_PORT")
	if port == "" {
		l.Fatalf("[http] port is empty")
	}
	addr := fmt.Sprintf("127.0.0.1:%s", port)
	server := &http.Server{
		Addr:         addr,
		ErrorLog:     l,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	return &httpHandler{
		logger: l,
		server: server,
	}
}

func (h *httpHandler) setRunnerHandler(rh *runnerHandler) {
	h.runnerHandler = rh
}

func (h *httpHandler) startHttpServer() {
	// Routes
	router := http.NewServeMux()
	router.HandleFunc("/ping", h.ping)
	router.HandleFunc("/status", h.status)
	h.server.Handler = router

	h.logger.Printf("[http] Server is listening '%s'", h.server.Addr)
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		h.logger.Fatalf("[http] Could not listen on %s: %v\n", h.server.Addr, err)
	}
}

func (h *httpHandler) shutdown() {
	h.logger.Println("[http] Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(HTTP_SERVER_SHUTDOWN_TIMEOUT))
	defer cancel()

	// server.SetKeepAlivesEnabled(false)
	if err := h.server.Shutdown(ctx); err != nil {
		h.logger.Printf("[http] Could not gracefully shutdown the server: %v\n", err)
	}
}

func (h *httpHandler) ping(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "pong")
}

func (h *httpHandler) status(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "goroutine num: %d", runtime.NumGoroutine())
}
