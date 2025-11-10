package neo4go

import (
	"fmt"
	"log"
	"os"
)

type defaultLogger struct {
	logger *log.Logger
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		logger: log.New(os.Stdout, "[neo4go] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(msg string, args ...any) {
	l.logger.Println("DEBUG:", l.formatMessage(msg, args...))
}

func (l *defaultLogger) Info(msg string, args ...any) {
	l.logger.Println("INFO:", l.formatMessage(msg, args...))
}

func (l *defaultLogger) Warn(msg string, args ...any) {
	l.logger.Println("WARN:", l.formatMessage(msg, args...))
}

func (l *defaultLogger) Error(msg string, args ...any) {
	l.logger.Println("ERROR:", l.formatMessage(msg, args...))
}

func (l *defaultLogger) formatMessage(msg string, args ...any) string {
	if len(args) == 0 {
		return msg
	}

	formatted := msg
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			formatted += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	return formatted
}
