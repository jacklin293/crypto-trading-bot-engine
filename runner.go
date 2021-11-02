package main

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/exchange"
	"crypto-trading-bot-engine/message"
	"crypto-trading-bot-engine/runner"
	"crypto-trading-bot-engine/strategy"
	"crypto-trading-bot-engine/strategy/contract"
	"crypto-trading-bot-engine/strategy/order"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/spf13/viper"
	"gorm.io/datatypes"
)

type runnerHandler struct {
	logger *log.Logger

	// Send the mark price to each runner channel of the same symbol
	runnersBySymbolMutex sync.RWMutex
	runnersBySymbolMap   map[string]map[string]*runner.ContractStrategyRunner

	// Strategy runner by strategy uuid
	runnerByUuidMap sync.Map // map[strategy.Uuid]chan runner.ContractStrategyRunner

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

	// This is the stop channel for handler itself
	eventsStopCh chan bool

	// There are channels for runner to communicate with handler
	eventsCh strategy.EventsCh

	// DB
	db *db.DB

	// For graceful shutdown, block until all STOP process of strategies have been completed
	blockWg sync.WaitGroup
}

func newRunnerHandler(l *log.Logger) *runnerHandler {
	return &runnerHandler{
		logger:                l,
		runnersBySymbolMap:    make(map[string]map[string]*runner.ContractStrategyRunner),
		symbolEntryTakenMutex: make(map[string]*sync.Mutex),
		exchangeUserMap:       make(map[string]exchange.Exchanger),
		eventsStopCh:          make(chan bool),
		eventsCh: strategy.EventsCh{
			Restart:   make(chan string),
			Enable:    make(chan string),
			Disable:   make(chan string),
			OutOfSync: make(chan string),
			Reset:     make(chan string),
		},
	}
}

func (h *runnerHandler) setLogger(l *log.Logger) {
	h.logger = l
}

func (h *runnerHandler) setDB(db *db.DB) {
	h.db = db
}

// NOTE only one event will be processed at a time.
// NOTE No need to worry about api spaming the event
func (h *runnerHandler) listenEvents() {
	for {
		select {
		case <-h.eventsStopCh:
			// Exit
			h.eventsStopCh <- true
			return

		// Enable/start a strategy
		case uuid := <-h.eventsCh.Enable:
			go h.enableContractStrategy(uuid)

		// Disable a strategy
		case uuid := <-h.eventsCh.Disable:
			go h.disableContractStrategy(uuid)

		// Process a strategy out of sync
		case uuid := <-h.eventsCh.OutOfSync:
			// Need to use goroutine, otherwise CheckPrice will get blocked as mutext can't be released
			go h.outOfSyncContractStrategy(uuid)

		// Reset a strategy
		case uuid := <-h.eventsCh.Reset:
			go h.resetContractStrategy(uuid)
		}
	}
}

func (h *runnerHandler) start() {
	// New sender
	h.newSender()

	go h.listenEvents()

	// Get enabled contract strategies
	contractStrategies, _, err := h.db.GetEnabledContractStrategies()
	if err != nil {
		h.logger.Fatal("err:", err)
	}

	for _, cs := range contractStrategies {
		user, err := h.getAndSetUserMap(cs.UserUuid)
		if err != nil {
			continue
		}
		h.startContractStrategyRunner(cs, user)

		// NOTE avoid sening too many messages at a time
		time.Sleep(time.Millisecond * 100)
	}
}

// NOTE FIXME Getting data from DB every time could make performance issue in the future
func (h *runnerHandler) getAndSetUserMap(uuid string) (*db.User, error) {
	user, err := h.db.GetUserByUuid(uuid)
	if err != nil {
		h.logger.Printf("[ERROR] user: '%s', err: %v\n", uuid, err)
		return user, err
	}
	// Set userMap by user uuid
	h.userMap.Store(uuid, user)
	return user, err
}

// NOTE Do not pass pointer of db.ContractStrategy because the last step of goroutine could be overriden by the next one in the loop
//      and causes 2 runnes for the same strategy
func (h *runnerHandler) startContractStrategyRunner(cs db.ContractStrategy, user *db.User) (err error) {
	if err = h.newContractStrategyRunner(&cs, user); err != nil {
		h.logger.Printf("[ERROR] strategy: '%s', user: '%s', symbol: '%s', err: %v\n", cs.Uuid, cs.UserUuid, cs.Symbol, err)

		// Disable the contract strategy
		// FIXME it's not in the map, so it didn't work
		h.eventsCh.OutOfSync <- cs.Uuid
		h.eventsCh.Disable <- cs.Uuid
		return
	}
	return nil
}

func (h *runnerHandler) newContractStrategyRunner(cs *db.ContractStrategy, user *db.User) error {
	r, err := runner.NewContractStrategyRunner(cs)
	if err != nil {
		return fmt.Errorf("Failed to new contract strategy runner, err: %v", err)
	}
	r.SetDB(h.db)
	r.SetLogger(h.logger)
	r.SetBeforeCloseFunc(h.stopContractStrategyRunner)
	r.SetHandlerBlockWg(&h.blockWg)
	r.SetHandlerEventsCh(&h.eventsCh)

	// New exchange client and set to the hook
	if err = h.newExchangeUserMap(cs.Exchange, user); err != nil {
		return err
	}
	r.SetExchangeForHook(h.exchangeUserMap[user.Uuid])

	// Set symbolEntryTakenMutex
	if h.symbolEntryTakenMutex[cs.UserUuid] == nil {
		h.symbolEntryTakenMutex[cs.UserUuid] = new(sync.Mutex)
	}
	r.SetSymbolEntryTakenMutexForHook(h.symbolEntryTakenMutex)

	// Set user for hook
	r.SetUser(user)

	// Set sender for contract strategy runner and hook
	r.SetSender(h.sender)

	// Manage contract strategy channel
	h.addIntoRunnersBySymbolMap(cs.Symbol, cs.Uuid, r)
	h.addIntoRunnerByUuidMap(cs.Uuid, r)

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

func (h *runnerHandler) newExchangeUserMap(name string, user *db.User) error {
	switch name {
	case "FTX":
		ex, err := exchange.NewExchange(viper.GetString("DEFAULT_EXCHANGE"), user.ExchangeApiKey)
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
	h.removeFromRunnersBySymbolMap(symbol, strategyUuid)
	h.removeFromRunnerByUuidMap(strategyUuid)
}

func (h *runnerHandler) addIntoRunnersBySymbolMap(symbol string, strategyUuid string, r *runner.ContractStrategyRunner) {
	h.runnersBySymbolMutex.Lock()
	defer h.runnersBySymbolMutex.Unlock()

	list, ok := h.runnersBySymbolMap[symbol]
	if ok {
		list[strategyUuid] = r
	} else {
		// If the key doesn't exist before, create one
		newList := make(map[string]*runner.ContractStrategyRunner)
		newList[strategyUuid] = r
		list = newList
	}
	h.runnersBySymbolMap[symbol] = list
}

// Remove contract strategy from runnersBySymbolMap
func (h *runnerHandler) removeFromRunnersBySymbolMap(symbol string, strategyUuid string) {
	h.runnersBySymbolMutex.Lock()
	defer h.runnersBySymbolMutex.Unlock()

	list, ok := h.runnersBySymbolMap[symbol]
	if ok {
		delete(list, strategyUuid)
	}
	if len(list) > 0 {
		h.runnersBySymbolMap[symbol] = list
	} else {
		// If there is no item in the key, remove the key from the map
		delete(h.runnersBySymbolMap, symbol)
	}
}

func (h *runnerHandler) addIntoRunnerByUuidMap(strategyUuid string, r *runner.ContractStrategyRunner) {
	h.runnerByUuidMap.Store(strategyUuid, r)
}

// Remove uuid from runnerByUuidMap
// NOTE: Delete doesn't actually delete the key from the internal map, it only deletes the value
//       The best way to check if the key is still available is to use Load()
func (h *runnerHandler) removeFromRunnerByUuidMap(strategyUuid string) {
	h.runnerByUuidMap.Delete(strategyUuid)
}

func (h *runnerHandler) stopAll() {
	// Send stop signal to each contract strategy runner
	h.runnerByUuidMap.Range(func(_, value interface{}) bool {
		r := value.(*runner.ContractStrategyRunner)
		r.Stop()
		return true
	})

	// Wait until everything in progress has been completed
	h.blockWg.Wait()

	// Make sure all strategies have been stopped then stop listening events
	h.eventsStopCh <- true
	<-h.eventsStopCh
}

func (h *runnerHandler) broadcastMark(symbol string, mark contract.Mark) {
	h.runnersBySymbolMutex.RLock()
	defer h.runnersBySymbolMutex.RUnlock()

	runners, ok := h.runnersBySymbolMap[symbol]
	if ok {
		for _, r := range runners {
			// NOTE Check this first, otherwise it might block ws handler due to no receiver if StopCh has been closed
			if r.CheckPriceEnabled {
				r.MarkCh <- mark
			}
		}
	}
}

func (h *runnerHandler) enableContractStrategy(uuid string) {
	// Make sure runner isn't in the list
	_, ok := h.runnerByUuidMap.Load(uuid)
	if ok {
		h.logger.Printf("[Error] enableContractStrategy - strategy '%s' is already in the map", uuid)
		return
	}

	// Get strategy from DB
	cs, err := h.db.GetContractStrategyByUuid(uuid)
	if err != nil {
		h.logger.Printf("[Error] enableContractStrategy - strategy '%s' not found", uuid)
		return
	}

	// Get user data
	user, err := h.getAndSetUserMap(cs.UserUuid)
	if err != nil {
		h.logger.Printf("[Error] enableContractStrategy - user '%s' not found", cs.UserUuid)
		return
	}

	// Enable contract strategy
	data := map[string]interface{}{
		"enabled": 1,
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] enableContractStrategy strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.TelegramChatId, text)
		return
	}

	// Start and new contract strategy runner
	if err := h.startContractStrategyRunner(*cs, user); err != nil {
		h.logger.Printf("[ERROR] enableContractStrategy strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please disable your strategy", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.TelegramChatId, text)
		return
	}

	h.logger.Printf("[Info] strategy: '%s', user: '%s', symbol: '%s' has been enabled", cs.Uuid, cs.UserUuid, cs.Symbol)
	text := fmt.Sprintf("[Info] '%s %s' has been enabled", order.TranslateSideByInt(cs.Side), cs.Symbol)
	h.sender.Send(user.TelegramChatId, text)
}

func (h *runnerHandler) disableContractStrategy(uuid string) {
	r, ok := h.runnerByUuidMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] disableContractStrategyCh - strategy '%s' isn't in the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	// NOTE Must be after WaitGroup, otherwise if runner receives stopCh and Wait()
	//      , it will cause `panic: sync: WaitGroup is reused before previous Wait has returned`
	//      , when trying to Add(1)
	r.(*runner.ContractStrategyRunner).RunnerMutex.Lock()
	defer r.(*runner.ContractStrategyRunner).RunnerMutex.Unlock()

	cs := r.(*runner.ContractStrategyRunner).ContractStrategy

	// Get user data
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] disableContractStrategyCh - user '%s' not found", cs.UserUuid)
		return
	}

	// Disable contract strategy
	data := map[string]interface{}{
		"enabled": 0,
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] disableContractStrategyCh strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
		return
	}

	r.(*runner.ContractStrategyRunner).Stop()

	h.logger.Printf("[Info] strategy: '%s', user: '%s', symbol: '%s' has been disabled", cs.Uuid, cs.UserUuid, cs.Symbol)
	text := fmt.Sprintf("[Info] '%s %s' has been disabled", order.TranslateSideByInt(cs.Side), cs.Symbol)
	h.sender.Send(user.(*db.User).TelegramChatId, text)
}

func (h *runnerHandler) outOfSyncContractStrategy(uuid string) {
	r, ok := h.runnerByUuidMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] outOfSyncContractStrategy - strategy '%s' isn't in the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	r.(*runner.ContractStrategyRunner).RunnerMutex.Lock()
	defer r.(*runner.ContractStrategyRunner).RunnerMutex.Unlock()

	// Get user data
	cs := r.(*runner.ContractStrategyRunner).ContractStrategy
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] outOfSyncContractStrategy - user '%s' not found", cs.UserUuid)
		return
	}

	// Change status
	data := map[string]interface{}{
		"position_status": int64(contract.UNKNOWN),
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] outOfSyncContractStrategy strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
		return
	}

	h.logger.Printf("[Warn] strategy: '%s', user: '%s', symbol: '%s' status has been changed to 'UNKNOWN'", cs.Uuid, cs.UserUuid, cs.Symbol)
	text := fmt.Sprintf("[Warn] '%s %s' is out of sync, please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
	h.sender.Send(user.(*db.User).TelegramChatId, text)
}

func (h *runnerHandler) resetContractStrategy(uuid string) {
	r, ok := h.runnerByUuidMap.Load(uuid)
	if !ok {
		h.logger.Printf("[Error] resetContractStrategy - strategy '%s' isn't in the map", uuid)
		return
	}

	// Block until finished
	r.(*runner.ContractStrategyRunner).RunnerBlockWg.Add(1)
	defer r.(*runner.ContractStrategyRunner).RunnerBlockWg.Done()

	r.(*runner.ContractStrategyRunner).RunnerMutex.Lock()
	defer r.(*runner.ContractStrategyRunner).RunnerMutex.Unlock()

	// Get strategy data
	cs := r.(*runner.ContractStrategyRunner).ContractStrategy

	// Get user data
	user, ok := h.userMap.Load(cs.UserUuid)
	if !ok {
		h.logger.Printf("[Error] resetContractStrategy - user '%s' not found", cs.UserUuid)
		return
	}

	// Reset status
	data := map[string]interface{}{
		"enabled":                 0,
		"position_status":         int64(contract.CLOSED),
		"exchange_orders_details": datatypes.JSONMap{},
	}
	if _, err := h.db.UpdateContractStrategy(cs.Uuid, data); err != nil {
		h.logger.Printf("[ERROR] resetContractStrategy strategy: '%s', user: '%s', symbol: '%s', err: %v", cs.Uuid, cs.UserUuid, cs.Symbol, err)
		text := fmt.Sprintf("[Error] '%s %s' Internal Server Error. Please check and reset your position and order", order.TranslateSideByInt(cs.Side), cs.Symbol)
		go h.sender.Send(user.(*db.User).TelegramChatId, text)
		return
	}

	r.(*runner.ContractStrategyRunner).Stop()

	h.logger.Printf("[Info] strategy: '%s', user: '%s', symbol: '%s' has been reset", cs.Uuid, cs.UserUuid, cs.Symbol)
	text := fmt.Sprintf("[Info] '%s %s' has been reset", order.TranslateSideByInt(cs.Side), cs.Symbol)
	h.sender.Send(user.(*db.User).TelegramChatId, text)
}
