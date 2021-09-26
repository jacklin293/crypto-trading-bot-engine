package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	SHUTDOWN_TIMEOUT = 300
)

// Gracefull shutdown
type signalHandler struct {
	closeRunner func()
	closeHttp   func()
	ctx         context.Context
	logger      *log.Logger
	sigCh       chan os.Signal
	doneCh      chan bool
}

func newSignalHandler(l *log.Logger) *signalHandler {
	ctx := context.Background()
	return &signalHandler{
		sigCh:  make(chan os.Signal, 1),
		ctx:    ctx,
		logger: l,
		doneCh: make(chan bool),
	}
}

func (h *signalHandler) setCloseRunnerFunc(f func()) {
	h.closeRunner = f
}

func (h *signalHandler) setCloseHttpFunc(f func()) {
	h.closeHttp = f
}

// Capture system signal
func (h *signalHandler) capture() {
	signal.Notify(h.sigCh, syscall.SIGINT, syscall.SIGTERM) // SIGINT=2, SIGTERM=15
	select {
	case <-h.sigCh:
		h.shutdown()
	}
}

func (h *signalHandler) shutdown() {
	h.logger.Printf("[pid:%d] terminating...\n", syscall.Getpid())
	h.closeHttp()
	h.closeRunner()

	var cancel context.CancelFunc
	if SHUTDOWN_TIMEOUT > 0 {
		h.ctx, cancel = context.WithTimeout(h.ctx, time.Duration(SHUTDOWN_TIMEOUT)*time.Second)
		defer cancel()
	}
	select {
	case <-h.doneCh:
	case <-h.ctx.Done():
		h.logger.Printf("Timeout: > %d seconds\n", SHUTDOWN_TIMEOUT)
	}
	h.logger.Printf("[pid:%d] terminated\n", syscall.Getpid())
}
