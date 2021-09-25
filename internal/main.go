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
	config := main.ServerConfig

	main.apiManager = api.New(config)

	main.server = http.New(
		main.apiManager,
		config,
	)
	main.server.Start()

	main.logger.Info().Msgf("serving streams from basedir %s: %s", config.BaseDir, config.Streams)
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
