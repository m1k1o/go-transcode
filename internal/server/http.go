package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ServerManagerCtx struct {
	logger zerolog.Logger
	config *Config
	router *chi.Mux
	server *http.Server
}

func New(config *Config) *ServerManagerCtx {
	logger := log.With().Str("module", "server").Logger()

	router := chi.NewRouter()
	router.Use(middleware.RequestID) // Create a request ID for each request

	// get real users ip
	if config.Proxy {
		router.Use(middleware.RealIP)
	}

	// add http logger
	router.Use(middleware.RequestLogger(&logformatter{logger}))
	router.Use(middleware.Recoverer) // Recover from panics without crashing server

	// serve static files
	if config.Static != "" {
		fs := http.FileServer(http.Dir(config.Static))
		router.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			if _, err := os.Stat(config.Static + r.RequestURI); os.IsNotExist(err) {
				http.StripPrefix(r.RequestURI, fs).ServeHTTP(w, r)
			} else {
				fs.ServeHTTP(w, r)
			}
		})
	}

	// mount pprof endpoint
	if config.PProf {
		withPProf(router)
		logger.Info().Msgf("with pprof endpoint at %s", pprofPath)
	}

	// use custom 404
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		//nolint
		_, _ = w.Write([]byte("404"))
	})

	return &ServerManagerCtx{
		logger: logger,
		config: config,
		router: router,
		server: &http.Server{
			Addr:    config.Bind,
			Handler: router,
		},
	}
}

func (s *ServerManagerCtx) Start() {
	if s.config.SSLCert != "" && s.config.SSLKey != "" {
		s.logger.Warn().Msg("TLS support is provided for convenience, but you should never use it in production. Use a reverse proxy (apache nginx caddy) instead!")
		go func() {
			if err := s.server.ListenAndServeTLS(s.config.SSLCert, s.config.SSLKey); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start https server")
			}
		}()
		s.logger.Info().Msgf("https listening on %s", s.server.Addr)
	} else {
		go func() {
			if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
				s.logger.Panic().Err(err).Msg("unable to start http server")
			}
		}()
		s.logger.Info().Msgf("http listening on %s", s.server.Addr)
	}
}

func (s *ServerManagerCtx) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *ServerManagerCtx) Mount(fn func(r *chi.Mux)) {
	fn(s.router)
}
