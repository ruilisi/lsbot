package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
)

// Level represents the logging level
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

var (
	currentLevel = LevelInfo
	mu           sync.RWMutex

	traceLogger *log.Logger
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	panicLogger *log.Logger
)

func init() {
	traceLogger = log.New(os.Stderr, "[TRACE] ", log.LstdFlags|log.Lshortfile)
	debugLogger = log.New(os.Stderr, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	infoLogger = log.New(os.Stderr, "", log.LstdFlags)
	warnLogger = log.New(os.Stderr, "[WARN] ", log.LstdFlags)
	errorLogger = log.New(os.Stderr, "[ERROR] ", log.LstdFlags)
	fatalLogger = log.New(os.Stderr, "[FATAL] ", log.LstdFlags)
	panicLogger = log.New(os.Stderr, "[PANIC] ", log.LstdFlags)
}

// ParseLevel parses a string into a Level
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	case "fatal":
		return LevelFatal, nil
	case "panic":
		return LevelPanic, nil
	default:
		return LevelInfo, fmt.Errorf("unknown log level: %s (use: trace, debug, info, warn, error, fatal, panic)", s)
	}
}

// SetLevel sets the global log level
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	currentLevel = level
}

// SetOutput redirects all logger output to w (e.g. to capture logs in mobile/gomobile context).
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	traceLogger.SetOutput(w)
	debugLogger.SetOutput(w)
	infoLogger.SetOutput(w)
	warnLogger.SetOutput(w)
	errorLogger.SetOutput(w)
	fatalLogger.SetOutput(w)
	panicLogger.SetOutput(w)
}

// GetLevel returns the current log level
func GetLevel() Level {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel
}

// Trace logs a message at trace level
func Trace(format string, v ...any) {
	mu.RLock()
	level := currentLevel
	mu.RUnlock()

	if level <= LevelTrace {
		traceLogger.Printf(format, v...)
	}
}

// Debug logs a message at debug level
func Debug(format string, v ...any) {
	mu.RLock()
	level := currentLevel
	mu.RUnlock()

	if level <= LevelDebug {
		debugLogger.Printf(format, v...)
	}
}

// Info logs a message at info level
func Info(format string, v ...any) {
	mu.RLock()
	level := currentLevel
	mu.RUnlock()

	if level <= LevelInfo {
		infoLogger.Printf(format, v...)
	}
}

// Warn logs a message at warn level
func Warn(format string, v ...any) {
	mu.RLock()
	level := currentLevel
	mu.RUnlock()

	if level <= LevelWarn {
		warnLogger.Printf(format, v...)
	}
}

// Error logs a message at error level
func Error(format string, v ...any) {
	mu.RLock()
	level := currentLevel
	mu.RUnlock()

	if level <= LevelError {
		errorLogger.Printf(format, v...)
	}
}

// Fatal logs a message at fatal level and exits
func Fatal(format string, v ...any) {
	fatalLogger.Fatalf(format, v...)
}

// Panic logs a message at panic level and panics
func Panic(format string, v ...any) {
	panicLogger.Panicf(format, v...)
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel <= LevelDebug
}

// IsTrace returns true if trace logging is enabled
func IsTrace() bool {
	mu.RLock()
	defer mu.RUnlock()
	return currentLevel <= LevelTrace
}
