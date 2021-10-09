package logger

import (
	"log"
	"os"
)

func NewLogger(env string, logPath string) (l *log.Logger) {
	l = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	if env == "prod" {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			log.Fatalf("failed to create log file, err: %v", err)
		}
		// NOTE Don't close it
		// defer f.Close()
		l.SetOutput(f)
	}
	return
}
