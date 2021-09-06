package utils

import (
	"strings"

	"github.com/rs/zerolog"
)

type LogWriterCtx struct {
	logger zerolog.Logger
}

func LogWriter(l zerolog.Logger) *LogWriterCtx {
	return &LogWriterCtx{
		logger: l,
	}
}

func (l LogWriterCtx) Write(p []byte) (n int, err error) {
	l.logger.Warn().Msg(strings.TrimSpace(string(p)))
	return len(p), nil
}
