package transcode

import (
	"os"
	"os/signal"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/m1k1o/go-transcode/internal/api"
	"github.com/m1k1o/go-transcode/internal/config"
	"github.com/m1k1o/go-transcode/internal/http"
)

var Service *Main

func init() {
	Service = &Main{
		RootConfig:   &config.Root{},
		ServerConfig: &config.Server{},
	}
}

type Main struct {
	RootConfig   *config.Root
	ServerConfig *config.Server

	logger     zerolog.Logger
	apiManager *api.ApiManagerCtx
	server     *http.ServerCtx
}

func (main *Main) Preflight() {
	main.logger = log.With().Str("service", "main").Logger()
}

func (main *Main) Start() {
	main.apiManager = api.New(main.ServerConfig)

	main.server = http.New(
		main.apiManager,
		main.ServerConfig,
	)
	main.server.Start()
}

func (main *Main) Shutdown() {
	if err := main.server.Shutdown(); err != nil {
		main.logger.Err(err).Msg("server shutdown with an error")
	} else {
		main.logger.Debug().Msg("server shutdown")
	}
}

func (main *Main) ServeCommand(cmd *cobra.Command, args []string) {
	main.logger.Info().Msg("starting main server")
	main.Start()
	main.logger.Info().Msg("main ready")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	sig := <-quit

	main.logger.Warn().Msgf("received %s, attempting graceful shutdown: \n", sig)
	main.Shutdown()
	main.logger.Info().Msg("shutdown complete")
}
