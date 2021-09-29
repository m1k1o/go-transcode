package http

import (
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi"
)

func (s *HttpManagerCtx) WithDebugPProf(pathPrefix string) {
	s.router.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", pprof.Index)

		r.Get("/{action}", func(w http.ResponseWriter, r *http.Request) {
			action := chi.URLParam(r, "action")

			switch action {
			case "cmdline":
				pprof.Cmdline(w, r)
			case "profile":
				pprof.Profile(w, r)
			case "symbol":
				pprof.Symbol(w, r)
			case "trace":
				pprof.Trace(w, r)
			default:
				pprof.Handler(action).ServeHTTP(w, r)
			}
		})
	})
}
