package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/m1k1o/go-transcode/internal/serve"
)

func init() {
	serve := serve.NewCommand()

	command := &cobra.Command{
		Use:   "serve",
		Short: "serve transcode server",
		Long:  `serve transcode server`,
		Run:   serve.Run,
	}

	onConfigLoad = append(onConfigLoad, func() {
		serve.Config.Set()
		serve.Preflight()
	})

	if err := serve.Config.Init(command); err != nil {
		log.Panic().Err(err).Msg("unable to run serve command")
	}

	rootCmd.AddCommand(command)
}
