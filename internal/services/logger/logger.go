package logger

import (
	"log"
	"os"
)

type Logger struct {
	Info  *log.Logger
	Error *log.Logger
}

func NewLogger() *Logger {

	infoFile, err := os.OpenFile("logs/info.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	errorFile, err := os.OpenFile("logs/error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	return &Logger{
		Info:  log.New(infoFile, "INFO: ", log.Ldate|log.Lmicroseconds|log.Llongfile),
		Error: log.New(errorFile, "ERROR: ", log.Ldate|log.Lmicroseconds|log.Llongfile),
	}
}
