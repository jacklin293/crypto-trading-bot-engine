package main

import (
	"crypto-trading-bot-main/db"
	"crypto-trading-bot-main/exchange"
	"crypto-trading-bot-main/message"
	"crypto-trading-bot-main/runner"
	"crypto-trading-bot-main/strategy/contract"
	"crypto-trading-bot-main/strategy/order"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gorm.io/datatypes"
)

type runnerHandler struct {
	logger *log.Logger

	// for sending the mark price to each runner channel of the same symbol
	symbolMarkChMutex sync.RWMutex
	symbolMarkChMap   map[string]map[string]chan contract.Mark

	// Strategy runner by strategy uuid
	runnerChMap sync.Map // map[strategy.Uuid]chan runner.ContractStrategyRunner

	// Check if entry order of symbol has been triggered already
	// One user can only trigger 1 entry order of symbol at once
	// NOTE: key is user_id
	symbolEntryTakenMutex map[string]*sync.Mutex

	// Exchange clients
	exchangeUserMap map[string]exchange.Exchanger // key is user_uuid

	// User data
	userMap sync.Map // map[userUuid]db.User

	// Message sender
	sender message.Messenger

	// Receive events from api
	apiEventCh chan []byte // json payload

	// This is the stop channel for handler itself
	eventStopCh chan bool

	// Disable contract strategy
	disableContractStrategyCh chan string // contract strategy uuid

	// Deal with contract strategy out of sync
	outOfSyncContractStrategyCh chan string // contract strategy uuid

	// Reset contract strategy
	resetContractStrategyCh chan string // contract strategy uuid

	// DB
	db *db.DB

	// For graceful shutdown, block until all STOP process of strategies have been completed
	blockWg sync.WaitGroup
}

func newRunnerHandler(l *log.Logger) *runnerHandler {
	return &runnerHandler{
		logger:                      l,
		symbolMarkChMap:             make(map[string]map[string]chan contract.Mark),
		symbolEntryTakenMutex:       make(map[string]*sync.Mutex),
		exchangeUserMap:             make(map[string]exchange.Exchanger),
		disableContractStrategyCh:   make(chan string),
		outOfSyncContractStrategyCh: make(chan string),
		resetContractStrategyCh:     make(chan string),
		eventStopCh:                 make(chan bool),
	}
}

func (h *runnerHandler) setLogger(l *log.Logger) {
	h.logger = l
}

func (h *runnerHandler) setDB(db *db.DB) {
	h.db = db
}

func (h *runnerHandler) process() {
	// New sender
	h.newSender()

	go h.listenEvents()

	// Get enabled contract strategies
	contractStrategies, _, err := h.db.GetEnabledContractStrategies()
	if err != nil {
		h.logger.Fatal("err:", err)
	}

	for _, cs := range contractStrategies {
		// Get user data
		user, err := h.setUserMap(cs.UserUuid)
		if err != nil {
			h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v\n", cs.Uuid, cs.UserUuid, cs.Symbol, err)
			continue
		}

		if err := h.newContractStrategyRunner(cs, user); err != nil {
			h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v\n", cs.Uuid, cs.UserUuid, cs.Symbol, err)

			// Disable the contract strategy
			h.outOfSyncContractStrategyCh <- cs.Uuid
			h.disableContractStrategyCh <- cs.Uuid
			continue
		}

		text := fmt.Sprintf("[Info] '%s %s' has been tracked (margin: $%s)", order.TranslateSideByInt(cs.Side), cs.Symbol, cs.Margin)
		go h.sender.Send(user.TelegramChatId, text)
	}
}

func (h *runnerHandler) newContractStrategyRunner(cs db.ContractStrategy, user db.User) error {
	if err := runner.ValidateExchangeOrdersDetails(&cs); err != nil {
		return fmt.Errorf("Check 'exchange_orders_details', err: %v", err)
	}

	r, err := runner.NewContractStrategyRunner(&cs)
	if err != nil {
		return fmt.Errorf("Failed to new contract strategy runner, err: %v", err)
	}
	r.SetDB(h.db)
	r.SetLogger(h.logger)
	r.SetBeforeCloseFunc(h.stopContractStrategyRunner)
	r.SetHandlerBlockWg(&h.blockWg)
	r.SetDisableCh(h.disableContractStrategyCh)
	r.SetOutOfSyncCh(h.outOfSyncContractStrategyCh)
	r.SetResetCh(h.resetContractStrategyCh)

	// New exchange client and set to the hook
	if err = h.newExchangeUserMap(cs.Exchange, &user); err != nil {
		return err
	}
	r.SetExchangeForHook(h.exchangeUserMap[user.Uuid])

	// Set symbolEntryTakenMutex
	if h.symbolEntryTakenMutex[cs.UserUuid] == nil {
		h.symbolEntryTakenMutex[cs.UserUuid] = new(sync.Mutex)
	}
	r.SetSymbolEntryTakenMutexForHook(h.symbolEntryTakenMutex)

	// Set user for hook
	r.SetUser(&user)

	// Set sender for contract strategy runner and hook
	r.SetSender(h.sender)

	// Manage contract strategy channel
	h.addIntoMarkChMap(cs.Symbol, cs.Uuid, r.MarkCh)
	h.addIntoRunnerChMap(cs.Uuid, r)

	go r.Run()
	return nil
}

func (h *runnerHandler) newSender() {
	data := map[string]interface{}{
		"token": viper.Get("TELEGRAM_TOKEN"),
	}
	sender, err := message.NewSender(viper.GetString("DEFAULT_SENDER_PLATFORM"), data)
	if err != nil {
		log.Fatal(err)
	}
	h.sender = sender
}

// NOTE FIXME Don't care about performance for now
func (h *runnerHandler) setUserMap(userUuid string) (db.User, error) {
	user, err := h.db.GetUserByUuid(userUuid)
	if err != nil {
		return db.User{}, fmt.Errorf("failed to get user, err: %v", err)
	}
	h.userMap.Store(userUuid, user)
	return *user, nil
}

func (h *runnerHandler) listenEvents() {
	for {
		select {
		case <-h.eventStopCh:
			// Exit
			h.eventStopCh <- true
			return

		case payload := <-h.apiEventCh:
			// TODO unmarshal event payload
			switch string(payload) {
			case "enable_strategy":
				// TODO close stopCh
				// TODO Read strategy from DB
				// TODO New strategy runner
				// TODO start that strategy runner again
			case "disable_strategy":
				// TODO Get strategy from DB
				// TODO Send to disableContractStrategyCh
			case "reset_strategy":
				// TODO Get strategy from DB
				// TODO Send to resetContractStrategyCh
			case "close_position":
			}

		// Disable a strategy
		case uuid := <-h.disableContractStrategyCh:
			h.disableContractStrategy(uuid)

			// NOTE workaround, please see the detail in another comment
			time.Sleep(time.Millisecond * 10)
			r, ok := h.runnerChMap.Load(uuid)
			if ok {
				close(r.(*runner.ContractStrategyRunner).StopCh)
			}

		// Process a strategy out of sync
		case uuid := <-h.outOfSyncContractStrategyCh:
			h.outOfSyncContractStrategy(uuid)

		// Reset a strategy
		case uuid := <-h.resetContractStrategyCh:
			h.resetContractStrategy(uuid)

			// NOTE In order to avoid `panic: close of closed channel`, sleep here a little to buy time for
			//      runner.beforeCloseFunc to clear the strategy from the map, so that it won't get strategy
			//      here (in the following step) and won't close closed channal again
			// Reproduction: send to this channel, then sleep 3 seconds at the end of `resetContractStrategy`
			//               , trigger system signal by pressing `ctrl+c` so that stopCh could be closed twice
			//               if there is no sleep blocking here
			time.Sleep(time.Millisecond * 10)
			r, ok := h.runnerChMap.Load(uuid)
			if ok {
				close(r.(*runner.ContractStrategyRunner).StopCh)
			}
		}
	}
}

func (h *runnerHandler) newExchangeUserMap(name string, user *db.User) error {
	switch name {
	case "FTX":
		ex, err := exchange.NewExchange(viper.GetString("DEFAULT_EXCHANGE"), user.ExchangeApiInfo)
		if err != nil {
			return fmt.Errorf("Failed to new exchange, err: %v", err)
		}
		// NOTE don't check if it exists, otherwise it won't get updated if api info changed
		h.exchangeUserMap[user.Uuid] = ex
	default:
		return fmt.Errorf("exchange '%s' not supported", name)
	}
	return nil
}

func (h *runnerHandler) stopContractStrategyRunner(symbol string, strategyUuid string) {
	h.removeFromMarkChMap(symbol, strategyUuid)
	h.removeFromRunnerChMap(strategyUuid)
}

func (h *runnerHandler) addIntoMarkChMap(symbol string, strategyUuid string, markCh chan contract.Mark) {
	h.symbolMarkChMutex.Lock()
	defer h.symbolMarkChMutex.Unlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		list[strategyUuid] = markCh
	} else {
		newList := make(map[string]chan contract.Mark)
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

func (h *runnerHandler) addIntoRunnerChMap(strategyUuid string, r *runner.ContractStrategyRunner) {
	h.runnerChMap.Store(strategyUuid, r)
}

// Remove uuid from runnerChMap
// NOTE: Delete doesn't actually delete the key from the internal map, it only deletes the value
//       The best way to check if the key is still available is to use Load()
func (h *runnerHandler) removeFromRunnerChMap(strategyUuid string) {
	h.runnerChMap.Delete(strategyUuid)
}

func (h *runnerHandler) stopAll() {
	// Send stop signal to each contract strategy runner
	h.runnerChMap.Range(func(_, value interface{}) bool {
		r := value.(*runner.ContractStrategyRunner)
		close(r.StopCh)
		return true
	})

	// Wait until everything in progress has been completed
	h.blockWg.Wait()

	// Make sure all strategies have been stopped then stop listening events
	h.eventStopCh <- true
	<-h.eventStopCh
}

func (h *runnerHandler) broadcastMark(symbol string, mark contract.Mark) {
	h.symbolMarkChMutex.RLock()
	defer h.symbolMarkChMutex.RUnlock()

	list, ok := h.symbolMarkChMap[symbol]
	if ok {
		for _, ch := range list {
			ch <- mark
		}
	}
}

func (h *runnerHandler) disableContractStrategy(uuid string) {
	r, ok := h.runnerChMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] disableContractStrategyCh - strategyUuid '%s' not found from the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	// Get user data
	cs := r.(*runner.ContractStrategyRunner).ContractStrategy
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] disableContractStrategyCh - userUuid '%s' not found from the map", uuid)
		return
	}

	// Disable contract strategy
	data := map[string]interface{}{
		"enabled": 0,
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		if ok {
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
			go h.sender.Send(user.(*db.User).TelegramChatId, text)
		}
		return
	}

	h.logger.Printf("[Info] strategy: '%s', user: '%s', symbol: '%s' has been disabled", cs.Uuid, cs.UserUuid, cs.Symbol)
	if ok {
		text := fmt.Sprintf("[Info] '%s %s' has been disabled", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
	}

	h.logger.Println("sleep 3s")
	time.Sleep(time.Second * 3)
}

func (h *runnerHandler) outOfSyncContractStrategy(uuid string) {
	r, ok := h.runnerChMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] disableContractStrategyCh - uuid '%s' not found from the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	// Get user data
	cs := r.(*runner.ContractStrategyRunner).ContractStrategy
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] userUuid '%s' not found from the map", uuid)
		return
	}

	// Change status
	data := map[string]interface{}{
		"position_status": int64(contract.UNKNOWN),
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		if ok {
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
			go h.sender.Send(user.(*db.User).TelegramChatId, text)
		}
		return
	}

	h.logger.Printf("[Warn] strategy: '%s', user: '%s', symbol: '%s' status has been changed to 'UNKNOWN'", cs.Uuid, cs.UserUuid, cs.Symbol)
	if ok {
		text := fmt.Sprintf("[Warn] '%s %s' is out of sync, please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
	}
}

func (h *runnerHandler) resetContractStrategy(uuid string) {
	r, ok := h.runnerChMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] strategyUuid '%s' not found from the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	// Get user data
	cs := r.(*runner.ContractStrategyRunner).ContractStrategy
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] userUuid '%s' not found from the map", uuid)
		return
	}

	// Reset status
	data := map[string]interface{}{
		"enabled":                 0,
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		if ok {
			text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
			go h.sender.Send(user.(*db.User).TelegramChatId, text)
		}
		return
	}

	h.logger.Printf("[Info] strategy: '%s', user: '%s', symbol: '%s' has been reset", cs.Uuid, cs.UserUuid, cs.Symbol)
	if ok {
		text := fmt.Sprintf("[Info] '%s %s' has been reset", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
	}
}
