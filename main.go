package main

import (
	"log"
	"os"
)

func main() {
	// logger
	l := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	// strategy
	strH := newStrategyHandler(l)
	go strH.process()

	// signal
	sigH := newSignalHandler(l)

	// websocket
	wsH := newWsHandler(l)
	wsH.setSignalDoneCh(sigH.doneCh)
	wsH.setStrategyHandler(strH)
	go wsH.connect()

	// signal
	sigH.setBeforeCloseFunc(wsH.close)
	sigH.capture()
}

// TODO Get pairs from redis, if not exists, read from DB
func getPairs() []string {
	return []string{"BTC-PERP"}
}
