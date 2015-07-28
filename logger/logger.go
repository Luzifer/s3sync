package logger

import "fmt"

// LogLevel defines a type for named log levels
type LogLevel uint

// Pre-Defined log levels to be used with this logging module
const (
	Error LogLevel = iota
	Warning
	Info
	Debug
)

// Logger is a wrapper around output to filter according to levels
type Logger struct {
	Level LogLevel
}

// New instanciates a new Logger and sets the preferred log level
func New(logLevel LogLevel) *Logger {
	return &Logger{
		Level: logLevel,
	}
}

// Log is the filtered equivalent to fmt.Println
func (l *Logger) Log(level LogLevel, line string) {
	if l.Level >= level {
		fmt.Println(line)
	}
}

// LogF is the filtered equivalent to fmt.Printf
func (l *Logger) LogF(level LogLevel, line string, args ...interface{}) {
	if l.Level >= level {
		fmt.Printf(line, args...)
	}
}

// ErrorF executes LogF with Error level
func (l *Logger) ErrorF(line string, args ...interface{}) {
	l.LogF(Error, line, args...)
}

// InfoF executes LogF with Info level
func (l *Logger) InfoF(line string, args ...interface{}) {
	l.LogF(Info, line, args...)
}

// DebugF executes LogF with Debug level
func (l *Logger) DebugF(line string, args ...interface{}) {
	l.LogF(Debug, line, args...)
}
