package main

import (
	"github.com/rs/zerolog/log"

	"m1k1o/transcode/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Panic().Err(err).Msg("failed to execute command")
	}
}
