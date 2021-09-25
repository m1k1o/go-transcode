package cmd

import (
	"os"
	"runtime"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	transcode "github.com/m1k1o/go-transcode/internal"
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
		config := transcode.Service.RootConfig
		config.Set()

		//////
		// logs
		//////
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

		if config.Debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		//////
		// configs
		//////
		if config.CfgFile != "" {
			viper.SetConfigFile(config.CfgFile) // use config file from the flag
		} else {
			if runtime.GOOS == "linux" {
				viper.AddConfigPath("/etc/transcode/")
			}

			viper.AddConfigPath(".")
			viper.SetConfigName("transcode")
		}

		viper.SetEnvPrefix("transcode")
		viper.AutomaticEnv() // read in environment variables that match

		err := viper.ReadInConfig()
		if err != nil && config.CfgFile != "" {
			log.Err(err)
		}

		logger := log.With().
			Bool("debug", config.Debug).
			Logger()

		file := viper.ConfigFileUsed()
		if file != "" {
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Info().Msg("config file reloaded")
				transcode.Service.ConfigReload()
			})

			viper.WatchConfig()
			logger.Info().Str("config", file).Msg("preflight complete with config file")
		} else {
			logger.Warn().Msg("preflight complete without config file")
		}
	})

	if err := transcode.Service.RootConfig.Init(root); err != nil {
		log.Panic().Err(err).Msg("unable to run root command")
	}
}
