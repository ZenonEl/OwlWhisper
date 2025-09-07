// Путь: cmd/fyne-gui/new-core/logger.go

package newcore

import (
	"log"
	"os"
)

// Уровни логирования
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

var (
	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	warnLogger  = log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Info выводит информационное сообщение.
func Info(format string, v ...interface{}) {
	infoLogger.Printf(format, v...)
}

// Warn выводит предупреждение.
func Warn(format string, v ...interface{}) {
	warnLogger.Printf(format, v...)
}

// Error выводит ошибку.
func Error(format string, v ...interface{}) {
	errorLogger.Printf(format, v...)
}
