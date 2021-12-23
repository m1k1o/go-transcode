package hlsproxy

import (
	"net/http"
	"time"
)

type Config struct {
	CacheCleanupPeriod time.Duration // how often should be cache cleanup called
	SegmentExpiration  time.Duration // how long should be segment kept in memory
	PlaylistExpiration time.Duration // how long should be playlist kept in memory
}

func (c Config) withDefaultValues() Config {
	if c.CacheCleanupPeriod == 0 {
		c.CacheCleanupPeriod = 4 * time.Second
	}
	if c.SegmentExpiration == 0 {
		c.SegmentExpiration = 60 * time.Second
	}
	if c.PlaylistExpiration == 0 {
		c.PlaylistExpiration = 1 * time.Second
	}
	return c
}

type Manager interface {
	Shutdown()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
