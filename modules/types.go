package modules

import "net/http"

type Config interface {
}

type Module interface {
	Shutdown()
	ConfigReload(config Config)
	Cleanup()
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
