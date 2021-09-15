package main

import (
	"crypto-trading-bot-main/strategy"
	"crypto-trading-bot-main/strategy/contract"
	"encoding/json"
	"log"
	"sync"
)

type strategyHandler struct {
	logger *log.Logger

	// for sending the mark price to each strategy channel of the same symbol
	symbolMarkChMutex sync.RWMutex
	symbolMarkChMap   map[string]map[string]chan strategy.Mark

	// Notify strategy to stop
	stopChMap sync.Map // map[strategy]chan StopCh

	// receive command from upstream handler
	eventCh chan string // e.g. {"action": "restart", "strategy_id": "TODO uuid or id?"}

	blockWg sync.WaitGroup
}

func newStrategyHandler(l *log.Logger) *strategyHandler {
	return &strategyHandler{
		logger:          l,
		symbolMarkChMap: make(map[string]map[string]chan strategy.Mark),
	}
}

func (h *strategyHandler) setLogger(l *log.Logger) {
	h.logger = l
}

func (h *strategyHandler) process() {
	// TODO check new enabled strategy
	strategyRows := []string{"fff", "bbb"}
	for _, strategyId := range strategyRows {
		payload := `
{
  "entry_type": "baseline",
  "entry_order": {
    "baseline_trigger": {
      "trigger_type": "line",
      "operator": "<=",
      "time_1": "2021-09-13 12:00:00",
      "price_1": 43370,
      "time_2": "2021-09-15 04:00:00",
      "price_2": 46682.32
    },
    "baseline_offset_percent": 0.005
  },
  "stop_loss_order": {
    "loss_tolerance_percent": 0.005,
    "baseline_readjustment_enabled": true
  },
  "take_profit_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": "<=",
      "price": 45342
    }
  }
}
`
		// TODO
		symbol := "BTC-PERP"
		status := contract.OPENED
		positionType := "long"

		var params map[string]interface{}
		err := json.Unmarshal([]byte(payload), &params)
		if err != nil {
			// TODO add strategy id into message
			h.logger.Println("Failed to unmarshal strategy params (id: TODO), err:", err)
		}
		s, err := strategy.NewStrategy(strategyId, symbol, positionType, status, params)
		if err != nil {
			// TODO add strategy id into message
			h.logger.Println("Failed to new strategy (id: TODO), err: ", err)
		}
		s.SetLogger(h.logger)
		s.SetBeforeCloseFunc(h.stopStrategyRunner)
		s.SetHandlerBlockWg(&h.blockWg)

		// Manage strategy channel
		h.addIntoMarhChMap(symbol, strategyId, s.MarkCh)
		h.addIntoStopChMap(strategyId, s.StopCh)

		go s.Run()
	}

	for {
		// TODO sub redis topic
		select {
		case action := <-h.eventCh:
			// TODO unmarshal event payload
			switch action {
			case "stop":
				// TODO close stopCh
			case "restart":
				// TODO close stopCh
				// TODO Read strategy from DB
				// TODO New strategy
				// TODO start that strategy again
			}
		}
	}
}

func (h *strategyHandler) addIntoMarhChMap(symbol string, strategyId string, markCh chan strategy.Mark) {
	h.symbolMarkChMutex.Lock()
	defer h.symbolMarkChMutex.Unlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		list[strategyId] = markCh
	} else {
		newList := make(map[string]chan strategy.Mark)
		newList[strategyId] = markCh
		list = newList
	}
	h.symbolMarkChMap[symbol] = list
}

func (h *strategyHandler) addIntoStopChMap(strategyId string, stopCh chan bool) {
	h.stopChMap.LoadOrStore(strategyId, stopCh)
}

func (h *strategyHandler) stopStrategyRunner(symbol string, strategyId string) {
	// Remove strategy from symbolMarkChMap
	h.symbolMarkChMutex.Lock()
	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		delete(list, strategyId)
	}
	if len(list) > 0 {
		h.symbolMarkChMap[symbol] = list
	} else {
		delete(h.symbolMarkChMap, symbol)
	}
	h.symbolMarkChMutex.Unlock()

	// Remove strategy from stopChMap
	// NOTE: Delete doesn't actually delete the key from the internal map, it only deletes the value
	//       The best way to check if the key is still available is to use Load()
	h.stopChMap.Delete(strategyId)
}

func (h *strategyHandler) stopAll() {
	// Send stop signal to each strategy runner
	h.stopChMap.Range(func(_, value interface{}) bool {
		ch := value.(chan bool)
		close(ch)
		return true
	})

	// Wait until everything in progress has been completed
	h.blockWg.Wait()
}

func (h *strategyHandler) broadcastMark(symbol string, mark strategy.Mark) {
	h.symbolMarkChMutex.RLock()
	defer h.symbolMarkChMutex.RUnlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		for _, ch := range list {
			ch <- mark
		}
	}
}
