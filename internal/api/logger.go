package api

import (
    "fmt"
    "strings"

	"github.com/rs/zerolog"
)

type LogWriter struct {
    logger zerolog.Logger
}

func NewLogWriter(l zerolog.Logger) *LogWriter {
    return &LogWriter{
		logger: l,
	}
}

func (l LogWriter) Write (p []byte) (n int, err error) {
	msg := fmt.Sprintf("%s", p)
	l.logger.Warn().Msg(strings.TrimSpace(msg))
    return len(p), nil
}
