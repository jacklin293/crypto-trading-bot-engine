package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
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
	router.HandleFunc("/strategy", h.strategy)
	router.HandleFunc("/list", h.list)
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

func (h *httpHandler) ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "pong")
}

func (h *httpHandler) status(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "goroutine num: %d", runtime.NumGoroutine())
}

func (h *httpHandler) list(w http.ResponseWriter, r *http.Request) {
	var uuids []string
	h.runnerHandler.runnerByUuidMap.Range(func(key, _ interface{}) bool {
		uuids = append(uuids, key.(string))
		return true
	})
	b, err := json.Marshal(uuids)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}
	fmt.Fprint(w, string(b))
}

func (h *httpHandler) strategy(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	action := strings.Trim(query.Get("action"), " ")
	uuid := strings.Trim(query.Get("uuid"), " ")
	h.logger.Printf("action: '%s', uuid: '%s'", action, uuid)

	if action == "" || uuid == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid params")
		return
	}

	switch action {
	case "restart":
		h.runnerHandler.eventsCh.Restart <- uuid
	case "enable":
		h.runnerHandler.eventsCh.Enable <- uuid
	case "disable":
		h.runnerHandler.eventsCh.Disable <- uuid
	case "close_position":
		// TODO
	default:
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "action '%s' not supported", action)
	}
}
