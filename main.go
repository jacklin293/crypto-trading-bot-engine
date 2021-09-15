package main

import (
	"log"
	"os"
)

func main() {
	// logger
	l := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	sh := newSignalHandler(l)
	wsh := newWsHandler(l, sh.doneCh)
	go wsh.connect()

	sh.setBeforeCloseFunc(wsh.close)
	sh.capture()
}

// TODO Get pairs from redis, if not exists, read from DB
func getPairs() []string {
	return []string{"BTC-PERP"}
}
