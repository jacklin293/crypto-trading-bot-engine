package main

import (
	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/runner"
	"crypto-trading-bot-main/strategy/contract"
	"fmt"
	"log"
	"os"
	"sync"
)

type runnerHandler struct {
	logger *log.Logger

	// for sending the mark price to each runner channel of the same symbol
	symbolMarkChMutex sync.RWMutex
	symbolMarkChMap   map[string]map[string]chan runner.Mark

	// Notify runner to stop
	stopChMap sync.Map // map[strategy.Uuid]chan StopCh

	// receive command from upstream handler
	eventCh chan string // e.g. {"action": "restart", "strategy_uuid": "strategy.Uuid"}

	// DB
	db *db.DB

	// For graceful shutdown, block until all STOP process of strategies have been completed
	blockWg sync.WaitGroup
}

func newRunnerHandler(l *log.Logger) *runnerHandler {
	return &runnerHandler{
		logger:          l,
		symbolMarkChMap: make(map[string]map[string]chan runner.Mark),
	}
}

func (h *runnerHandler) setLogger(l *log.Logger) {
	h.logger = l
}

func (h *runnerHandler) setDB(db *db.DB) {
	h.db = db
}

func (h *runnerHandler) process() {
	contractStrategies, _, err := h.db.GetEnabledContractStrategies()
	if err != nil {
		h.logger.Println("err:", err)
		os.Exit(1)
	}

	for _, cs := range contractStrategies {
		if err := h.newContractStrategyRunner(cs); err != nil {
			h.logger.Println("err:", err)
		}
	}

	for {
		// TODO sub redis topic
		select {
		case action := <-h.eventCh:
			// TODO unmarshal event payload
			switch action {
			case "stop_contract_strategy_runner":
				// TODO close stopCh
			case "restart_contract_strategy_runner":
				// TODO close stopCh
				// TODO Read strategy from DB
				// TODO New strategy runner
				// TODO start that strategy runner again
			case "start_contract_strategy_runner":
			}
		}
	}
}

func (h *runnerHandler) newContractStrategyRunner(cs db.ContractStrategy) error {
	r, err := runner.NewContractStrategyRunner(cs)
	if err != nil {
		return fmt.Errorf("Failed to new contract strategy runner '%s', err: %s\n", cs.Uuid, err)
	}
	r.SetPositionStatus(contract.Status(cs.PositionStatus))
	r.SetDB(h.db)
	r.SetLogger(h.logger)
	r.SetBeforeCloseFunc(h.stopContractStrategyRunner)
	r.SetHandlerBlockWg(&h.blockWg)

	// Manage contract strategy channel
	h.addIntoMarkChMap(cs.Symbol, cs.Uuid, r.MarkCh)
	h.addIntoStopChMap(cs.Uuid, r.StopCh)

	go r.Run()
	return nil
}

func (h *runnerHandler) stopContractStrategyRunner(symbol string, strategyUuid string) {
	h.removeFromMarkChMap(symbol, strategyUuid)
	h.removeFromStopChMap(strategyUuid)
}

func (h *runnerHandler) addIntoMarkChMap(symbol string, strategyUuid string, markCh chan runner.Mark) {
	h.symbolMarkChMutex.Lock()
	defer h.symbolMarkChMutex.Unlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		list[strategyUuid] = markCh
	} else {
		newList := make(map[string]chan runner.Mark)
		newList[strategyUuid] = markCh
		list = newList
	}
	h.symbolMarkChMap[symbol] = list
}

// Remove contract strategy from symbolMarkChMap
func (h *runnerHandler) removeFromMarkChMap(symbol string, strategyUuid string) {
	h.symbolMarkChMutex.Lock()
	defer h.symbolMarkChMutex.Unlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		delete(list, strategyUuid)
	}
	if len(list) > 0 {
		h.symbolMarkChMap[symbol] = list
	} else {
		delete(h.symbolMarkChMap, symbol)
	}
}

func (h *runnerHandler) addIntoStopChMap(strategyUuid string, stopCh chan bool) {
	h.stopChMap.LoadOrStore(strategyUuid, stopCh)
}

// Remove contract strategy from stopChMap
// NOTE: Delete doesn't actually delete the key from the internal map, it only deletes the value
//       The best way to check if the key is still available is to use Load()
func (h *runnerHandler) removeFromStopChMap(strategyUuid string) {
	h.stopChMap.Delete(strategyUuid)
}

func (h *runnerHandler) stopAll() {
	// Send stop signal to each contract strategy runner
	h.stopChMap.Range(func(_, value interface{}) bool {
		ch := value.(chan bool)
		close(ch)
		return true
	})

	// Wait until everything in progress has been completed
	h.blockWg.Wait()
}

func (h *runnerHandler) broadcastMark(symbol string, mark runner.Mark) {
	h.symbolMarkChMutex.RLock()
	defer h.symbolMarkChMutex.RUnlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		for _, ch := range list {
			ch <- mark
		}
	}
}
