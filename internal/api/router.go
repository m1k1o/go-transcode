package api

import (
	"net/http"

	"github.com/go-chi/chi"
)

type ApiManagerCtx struct {}

func New() *ApiManagerCtx {

	return &ApiManagerCtx{}
}

func (a *ApiManagerCtx) Mount(r *chi.Mux) {
	r.Get("/ping", func (w http.ResponseWriter, r *http.Request) {
		//nolint
		w.Write([]byte("pong"))
	})
}
