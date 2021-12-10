package hlsproxy

import "net/http"

type Manager interface {
	Shutdown()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
