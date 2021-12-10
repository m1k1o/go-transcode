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

type Manager interface {
	Shutdown()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
