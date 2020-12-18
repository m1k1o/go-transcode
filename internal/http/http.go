package http

import (
	"context"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"m1k1o/transcode/internal/types"
	"m1k1o/transcode/internal/config"
)

type ServerCtx struct {
	logger zerolog.Logger
	router *chi.Mux
	http   *http.Server
	conf   *config.Server
}

func New(ApiManager types.ApiManager, conf *config.Server) *ServerCtx {
	logger := log.With().Str("module", "http").Logger()

	router := chi.NewRouter()
	router.Use(middleware.Recoverer) // Recover from panics without crashing server
	router.Use(middleware.RequestID) // Create a request ID for each request
	router.Use(Logger) // Log API request calls using custom logger function

	ApiManager.Mount(router)

	if conf.Static != "" {
		fs := http.FileServer(http.Dir(conf.Static))
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat(conf.Static + r.RequestURI); os.IsNotExist(err) {
				http.StripPrefix(r.RequestURI, fs).ServeHTTP(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
		})
	}

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		//nolint
		w.Write([]byte("404"))
	})

	http := &http.Server{
		Addr:    conf.Bind,
		Handler: router,
	}

	return &ServerCtx{
		logger: logger,
		router: router,
		http:   http,
		conf:   conf,
	}
}

func (s *ServerCtx) Start() {
	if s.conf.Cert != "" && s.conf.Key != "" {
		go func() {
			if err := s.http.ListenAndServeTLS(s.conf.Cert, s.conf.Key); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start https server")
			}
		}()
		s.logger.Info().Msgf("https listening on %s", s.http.Addr)
	} else {
		go func() {
			if err := s.http.ListenAndServe(); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start http server")
			}
		}()
		s.logger.Info().Msgf("http listening on %s", s.http.Addr)
	}
}

func (s *ServerCtx) Shutdown() error {
	return s.http.Shutdown(context.Background())
}
