package logger

import "fmt"

type LogLevel uint

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

func New(logLevel LogLevel) *Logger {
	return &Logger{
		Level: logLevel,
	}
}

func (l *Logger) Log(level LogLevel, line string) {
	if l.Level >= level {
		fmt.Println(line)
	}
}

func (l *Logger) LogF(level LogLevel, line string, args ...interface{}) {
	if l.Level >= level {
		fmt.Printf(line, args...)
	}
}

func (l *Logger) ErrorF(line string, args ...interface{}) {
	l.LogF(Error, line, args...)
}

func (l *Logger) InfoF(line string, args ...interface{}) {
	l.LogF(Info, line, args...)
}

func (l *Logger) DebugF(line string, args ...interface{}) {
	l.LogF(Debug, line, args...)
}
