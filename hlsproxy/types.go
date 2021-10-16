package hlsproxy

import "net/http"

type Manager interface {
	Start() error
	Stop()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
