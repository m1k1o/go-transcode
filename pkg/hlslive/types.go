package hlslive

import (
	"net/http"
	"time"
)

type Config struct {
	CleanupPeriod       time.Duration // how often should be cleanup called
	PlaylistTimeout     time.Duration // timeout for first playlist, when it waits for new data
	HlsMinimumSegments  int           // minimum segments available to consider stream as active
	ActiveIdleTimeout   time.Duration // how long must be active stream idle to be considered as dead
	InactiveIdleTimeout time.Duration // how long must be iactive stream idle to be considered as dead
}

func (c Config) withDefaultValues() Config {
	if c.CleanupPeriod == 0 {
		c.CleanupPeriod = 4 * time.Second
	}
	if c.PlaylistTimeout == 0 {
		c.PlaylistTimeout = 60 * time.Second
	}
	if c.HlsMinimumSegments == 0 {
		c.HlsMinimumSegments = 2
	}
	if c.ActiveIdleTimeout == 0 {
		c.ActiveIdleTimeout = 12 * time.Second
	}
	if c.InactiveIdleTimeout == 0 {
		c.InactiveIdleTimeout = 24 * time.Second
	}
	return c
}

type Manager interface {
	Start() error
	Stop()
	Cleanup()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)

	OnStart(event func())
	OnCmdLog(event func(message string))
	OnStop(event func(err error))
}
