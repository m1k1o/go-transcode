package cmd

import (
	"os"
	"runtime"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/m1k1o/go-transcode/internal"
)

func Execute() error {
	return root.Execute()
}

var root = &cobra.Command{
	Use:   "transcode",
	Short: "transcode server",
	Long:  `transcode server`,
}

func init() {
	cobra.OnInitialize(func() {
		//////
		// logs
		//////
		zerolog.TimeFieldFormat = ""
		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		if viper.GetBool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

		//////
		// configs
		//////
		config := viper.GetString("config")
		if config != "" {
			viper.SetConfigFile(config) // Use config file from the flag.
		} else {
			if runtime.GOOS == "linux" {
				viper.AddConfigPath("/etc/transcode/")
			}

			viper.AddConfigPath(".")
			viper.SetConfigName("transcode")
		}

		viper.SetEnvPrefix("transcode")
		viper.AutomaticEnv() // read in environment variables that match

		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				log.Error().Err(err)
			}
			if config != "" {
				log.Error().Err(err)
			}
		}

		file := viper.ConfigFileUsed()
		logger := log.With().
			Bool("debug", viper.GetBool("debug")).
			Str("logging", viper.GetString("logs")).
			Str("config", file).
			Logger()

		if file == "" {
			logger.Warn().Msg("preflight complete without config file")
		} else {
			logger.Info().Msg("preflight complete")
		}

		transcode.Service.RootConfig.Set()
	})

	if err := transcode.Service.RootConfig.Init(root); err != nil {
		log.Panic().Err(err).Msg("unable to run root command")
	}
}
