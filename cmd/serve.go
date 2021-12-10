package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	transcode "github.com/m1k1o/go-transcode/internal"
)

func init() {
	command := &cobra.Command{
		Use:   "serve",
		Short: "serve transcode server",
		Long:  `serve transcode server`,
		Run:   transcode.Service.ServeCommand,
	}

	onConfigLoad = append(onConfigLoad, func() {
		transcode.Service.ServerConfig.Set()
		transcode.Service.Preflight()
	})

	if err := transcode.Service.ServerConfig.Init(command); err != nil {
		log.Panic().Err(err).Msg("unable to run serve command")
	}

	rootCmd.AddCommand(command)
}
