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
