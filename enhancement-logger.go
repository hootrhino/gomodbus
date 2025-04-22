package modbus

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// LogLevel type defines the severity of a log message.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelNone // Disables logging
)

// LevelToString maps LogLevel to its string representation.
var LevelToString = map[LogLevel]string{
	LevelDebug:   "DEBUG",
	LevelInfo:    "INFO",
	LevelWarning: "WARNING",
	LevelError:   "ERROR",
	LevelNone:    "NONE",
}

// StringToLevel maps string representation of LogLevel to its value.
var StringToLevel = map[string]LogLevel{
	"DEBUG":   LevelDebug,
	"INFO":    LevelInfo,
	"WARNING": LevelWarning,
	"ERROR":   LevelError,
	"NONE":    LevelNone,
}

// SimpleLogger implements io.WriteCloser and supports setting log level.
type SimpleLogger struct {
	mu         sync.Mutex
	level      LogLevel
	output     io.WriteCloser
	timeFormat string
	prefix     string
}

// NewSimpleLogger creates a new SimpleLogger instance.
// If output is nil, it defaults to os.Stdout.
func NewSimpleLogger(output io.WriteCloser, level LogLevel, prefix string) *SimpleLogger {
	if output == nil {
		output = os.Stdout
	}
	return &SimpleLogger{
		level:      level,
		output:     output,
		timeFormat: time.RFC3339,
		prefix:     prefix,
	}
}

// SetLevel sets the logging level of the SimpleLogger.
func (l *SimpleLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current logging level of the SimpleLogger.
func (l *SimpleLogger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetLevelFromString sets the logging level from a string representation (e.g., "DEBUG").
func (l *SimpleLogger) SetLevelFromString(levelStr string) error {
	levelStrUpper := strings.ToUpper(levelStr)
	if level, ok := StringToLevel[levelStrUpper]; ok {
		l.SetLevel(level)
		return nil
	}
	return fmt.Errorf("invalid log level: %s. Available levels: %v", levelStr, getAvailableLevels())
}

func getAvailableLevels() []string {
	levels := make([]string, 0, len(StringToLevel))
	for levelStr := range StringToLevel {
		levels = append(levels, levelStr)
	}
	return levels
}

// Write implements the io.Writer interface. It filters log messages based on the set level.
// The input 'p' is expected to be a log message string without level prefix.
func (l *SimpleLogger) Write(p []byte) (n int, err error) {
	message := string(p)
	level := determineLevel(message)

	if level >= l.GetLevel() && l.GetLevel() != LevelNone {
		l.mu.Lock()
		defer l.mu.Unlock()
		timestamp := time.Now().Format(l.timeFormat)
		levelStr := LevelToString[level]
		formattedMessage := fmt.Sprintf("%s [%s] <%s> %s", timestamp, levelStr, l.prefix, strings.TrimSpace(message))
		return l.output.Write([]byte(formattedMessage + "\n"))
	}
	return len(p), nil
}

// Close implements the io.Closer interface. It closes the underlying output if it's not os.Stdout.
func (l *SimpleLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if closer, ok := l.output.(io.Closer); ok && l.output != os.Stdout {
		return closer.Close()
	}
	return nil
}

// determineLevel tries to infer the log level from the message prefix.
// If no known prefix is found, it defaults to LevelInfo.
func determineLevel(message string) LogLevel {
	upperMessage := strings.ToUpper(message)
	if strings.HasPrefix(upperMessage, "[DEBUG]") ||
		strings.HasPrefix(upperMessage, "DEBUG:") {
		return LevelDebug
	}
	if strings.HasPrefix(upperMessage, "[INFO]") ||
		strings.HasPrefix(upperMessage, "INFO:") {
		return LevelInfo
	}
	if strings.HasPrefix(upperMessage, "[WARNING]") ||
		strings.HasPrefix(upperMessage, "WARN:") ||
		strings.HasPrefix(upperMessage, "WARNING:") {
		return LevelWarning
	}
	if strings.HasPrefix(upperMessage, "[ERROR]") ||
		strings.HasPrefix(upperMessage, "ERROR:") {
		return LevelError
	}
	return LevelInfo // Default level if no prefix is found
}
