package main

import (
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/go-transcode/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Panic().Err(err).Msg("failed to execute command")
	}
}
