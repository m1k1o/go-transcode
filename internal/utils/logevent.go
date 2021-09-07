package utils

import "strings"

type LogEventCtx struct {
	event func(message string)
}

func LogEvent(event func(message string)) *LogEventCtx {
	return &LogEventCtx{
		event: event,
	}
}

func (l LogEventCtx) Write(p []byte) (n int, err error) {
	l.event(strings.TrimSpace(string(p)))
	return len(p), nil
}
