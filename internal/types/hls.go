package types

import "net/http"

type HlsManager interface {
	Start() error
	Stop()
	Cleanup()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
