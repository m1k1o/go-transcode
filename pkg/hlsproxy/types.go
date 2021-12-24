package hlsproxy

import (
	"net/http"
	"strings"
	"time"
)

type Config struct {
	PlaylistBaseUrl    string
	PlaylistPathPrefix string
	SegmentBaseUrl     string // optional: will be used playlist value if empty
	SegmentPathPrefix  string // optional: will be used playlist value if empty

	CacheCleanupPeriod time.Duration // how often should be cache cleanup called
	SegmentExpiration  time.Duration // how long should be segment kept in memory
	PlaylistExpiration time.Duration // how long should be playlist kept in memory
}

func (c Config) withDefaultValues() Config {
	if c.SegmentBaseUrl == "" {
		c.SegmentBaseUrl = c.PlaylistBaseUrl
	}
	if c.SegmentPathPrefix == "" {
		c.SegmentPathPrefix = c.PlaylistPathPrefix
	}
	if c.CacheCleanupPeriod == 0 {
		c.CacheCleanupPeriod = 4 * time.Second
	}
	if c.SegmentExpiration == 0 {
		c.SegmentExpiration = 60 * time.Second
	}
	if c.PlaylistExpiration == 0 {
		c.PlaylistExpiration = 1 * time.Second
	}
	// ensure it ends with single /
	c.PlaylistBaseUrl = strings.TrimRight(c.PlaylistBaseUrl, "/") + "/"
	c.SegmentBaseUrl = strings.TrimRight(c.SegmentBaseUrl, "/") + "/"
	// ensure it starts and ends with single /
	c.PlaylistPathPrefix = "/" + strings.Trim(c.PlaylistPathPrefix, "/") + "/"
	c.SegmentPathPrefix = "/" + strings.Trim(c.SegmentPathPrefix, "/") + "/"
	return c
}

type Manager interface {
	Shutdown()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeSegment(w http.ResponseWriter, r *http.Request)
}
