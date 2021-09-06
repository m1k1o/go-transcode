package hls

import "net/http"

type Manager interface {
	Start() error
	Stop()
	Cleanup()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
