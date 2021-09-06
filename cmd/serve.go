package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/m1k1o/go-transcode"
	"github.com/m1k1o/go-transcode/internal/config"
)

func init() {
	command := &cobra.Command{
		Use:   "serve",
		Short: "serve transcode server",
		Long:  `serve transcode server`,
		Run:   transcode.Service.ServeCommand,
	}

	configs := []config.Config{
		transcode.Service.ServerConfig,
	}

	cobra.OnInitialize(func() {
		for _, cfg := range configs {
			cfg.Set()
		}
		transcode.Service.Preflight()
	})

	for _, cfg := range configs {
		if err := cfg.Init(command); err != nil {
			log.Panic().Err(err).Msg("unable to run serve command")
		}
	}

	root.AddCommand(command)
}
