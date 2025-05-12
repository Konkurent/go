package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

func init() {
	// Создаем директорию для логов, если она не существует
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatal("Failed to create log directory:", err)
	}

	// Открываем файлы для логирования
	infoFile, err := os.OpenFile(filepath.Join(logDir, "info.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open info log file:", err)
	}

	errorFile, err := os.OpenFile(filepath.Join(logDir, "error.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open error log file:", err)
	}

	debugFile, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open debug log file:", err)
	}

	// Инициализируем логгеры
	InfoLogger = log.New(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	DebugLogger = log.New(debugFile, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// LogInfo логирует информационное сообщение
func LogInfo(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	InfoLogger.Printf("%s:%d - %s", filepath.Base(file), line, fmt.Sprintf(format, v...))
}

// LogError логирует сообщение об ошибке
func LogError(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	ErrorLogger.Printf("%s:%d - %s", filepath.Base(file), line, fmt.Sprintf(format, v...))
}

// LogDebug логирует отладочное сообщение
func LogDebug(format string, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	DebugLogger.Printf("%s:%d - %s", filepath.Base(file), line, fmt.Sprintf(format, v...))
}

// LogOperation логирует операцию с метриками
func LogOperation(operation string, startTime time.Time, err error) {
	duration := time.Since(startTime)
	if err != nil {
		LogError("Operation %s failed after %v: %v", operation, duration, err)
	} else {
		LogInfo("Operation %s completed in %v", operation, duration)
	}
}
