package main

import (
	"crypto-trading-bot-main/db"
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

func main() {
	// Read config
	loadConfig()

	// Connect to DB
	db, err := db.NewDB(viper.GetString("DB_DSN"))
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
	wsh.setDB(db)
	go wsh.connect()

	// signal
	sh.setBeforeCloseFunc(wsh.close)
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
