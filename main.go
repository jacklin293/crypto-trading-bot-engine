package main

import (
	"crypto-trading-bot-main/db"
	"log"
	"os"
)

const (
	DB_DSN = "root:root@tcp(127.0.0.1:3306)/crypto?charset=utf8mb4&parseTime=true"
)

func main() {
	// Connect to DB
	db, err := db.NewDB(DB_DSN)
	if err != nil {
		log.Fatal(err)
	}

	// logger
	l := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)

	// runner
	srh := newRunnerHandler(l)
	srh.setDB(db)
	go srh.process()

	// signal
	sh := newSignalHandler(l)

	// websocket
	wsh := newWsHandler(l)
	wsh.setSignalDoneCh(sh.doneCh)
	wsh.setRunnerHandler(srh)
	go wsh.connect()

	// signal
	sh.setBeforeCloseFunc(wsh.close)
	sh.capture()
}

// TODO Get pairs from redis, if not exists, read from DB
func getPairs() []string {
	return []string{"BTC-PERP"}
}
