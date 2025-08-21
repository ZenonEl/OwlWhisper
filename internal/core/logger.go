package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel представляет уровень логирования
type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// LogOutput определяет куда выводить логи
type LogOutput int

const (
	LogOutputNone LogOutput = iota
	LogOutputConsole
	LogOutputFile
	LogOutputBoth
)

// Logger управляет логированием
type Logger struct {
	level   LogLevel
	output  LogOutput
	file    *os.File
	console io.Writer
	mu      sync.RWMutex
	prefix  string
}

// NewLogger создает новый логгер
func NewLogger(level LogLevel, output LogOutput, logDir string) (*Logger, error) {
	l := &Logger{
		level:   level,
		output:  output,
		console: os.Stdout,
	}

	// Если нужен вывод в файл, создаем его
	if output == LogOutputFile || output == LogOutputBoth {
		if logDir == "" {
			logDir = "logs"
		}

		// Создаем директорию если не существует
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("не удалось создать директорию логов: %v", err)
		}

		logFile := filepath.Join(logDir, "owlwhisper.log")
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("не удалось открыть файл логов: %v", err)
		}
		l.file = file
	}

	return l, nil
}

// SetLevel устанавливает уровень логирования
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetOutput изменяет настройки вывода
func (l *Logger) SetOutput(output LogOutput) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// write записывает сообщение в нужные места
func (l *Logger) write(level LogLevel, format string, args ...interface{}) {
	if l.level < level {
		return
	}

	message := fmt.Sprintf(format, args...)
	timestamp := fmt.Sprintf("[%s] ", getTimestamp())
	fullMessage := timestamp + message + "\n"

	l.mu.RLock()
	defer l.mu.RUnlock()

	switch l.output {
	case LogOutputConsole:
		fmt.Fprint(l.console, fullMessage)
	case LogOutputFile:
		if l.file != nil {
			fmt.Fprint(l.file, fullMessage)
		}
	case LogOutputBoth:
		fmt.Fprint(l.console, fullMessage)
		if l.file != nil {
			fmt.Fprint(l.file, fullMessage)
		}
	}
}

// Debug логирует отладочное сообщение
func (l *Logger) Debug(format string, args ...interface{}) {
	l.write(LogLevelDebug, "[DEBUG] "+format, args...)
}

// Info логирует информационное сообщение
func (l *Logger) Info(format string, args ...interface{}) {
	l.write(LogLevelInfo, "[INFO] "+format, args...)
}

// Warn логирует предупреждение
func (l *Logger) Warn(format string, args ...interface{}) {
	l.write(LogLevelWarn, "[WARN] "+format, args...)
}

// Error логирует ошибку
func (l *Logger) Error(format string, args ...interface{}) {
	l.write(LogLevelError, "[ERROR] "+format, args...)
}

// Close закрывает файл логов
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// getTimestamp возвращает текущее время в нужном формате
func getTimestamp() string {
	// Используем простой формат времени для производительности
	return time.Now().Format("2006/01/02 15:04:05")
}

// Глобальный логгер по умолчанию
var globalLogger *Logger

// InitGlobalLogger инициализирует глобальный логгер
func InitGlobalLogger(level LogLevel, output LogOutput, logDir string) error {
	var err error
	globalLogger, err = NewLogger(level, output, logDir)
	return err
}

// GetGlobalLogger возвращает глобальный логгер
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		// Создаем логгер по умолчанию если не инициализирован
		globalLogger, _ = NewLogger(LogLevelInfo, LogOutputConsole, "")
	}
	return globalLogger
}

// Глобальные функции для удобства
func Debug(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warn(format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, args...)
	}
}
