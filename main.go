package main

import (
	"crypto-trading-bot-engine/db"
	"crypto-trading-bot-engine/util/logger"
	"fmt"

	"github.com/spf13/viper"
)

func main() {
	// Read config
	loadConfig()

	// logger
	l := logger.NewLogger(viper.GetString("ENV"), viper.GetString("LOG_PATH"))

	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
	if err != nil {
		l.Fatal(err)
	}

	// runner
	rh := newRunnerHandler(l)
	rh.setDB(db)
	go rh.start()

	// signal
	sh := newSignalHandler(l)

	// websocket
	wsh := newWsHandler(l)
	wsh.setSignalDoneCh(sh.doneCh)
	wsh.setRunnerHandler(rh)
	wsh.setDB(db)
	go wsh.connect()

	// http
	hh := newHttpHandler(l)
	hh.setRunnerHandler(rh)
	go hh.startHttpServer()

	// signal
	sh.setCloseHttpFunc(hh.shutdown)
	sh.setCloseRunnerFunc(wsh.close)
	sh.capture()
}

func loadConfig() {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	err := viper.ReadInConfig()   // Find and read the config file
	if err != nil {               // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
	fmt.Printf("Load config (ENV: %s)\n", viper.Get("ENV"))
}
