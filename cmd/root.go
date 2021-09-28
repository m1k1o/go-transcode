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
		//////
		// logs
		//////
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

		//////
		// configs
		//////

		// at this point we did not read any config data, so we need to tell
		// explicitly how to get this value
		cfgFile := viper.GetString("config") // Use config file from the flag
		if cfgFile == "" {
			cfgFile = os.Getenv("TRANSCODE_CONFIG") // Use config file from the env
		}

		if cfgFile != "" {
			viper.SetConfigFile(cfgFile) // use config file from the flag
		} else {
			if runtime.GOOS == "linux" {
				viper.AddConfigPath("/etc/transcode/")
			}

			viper.AddConfigPath(".")
			viper.SetConfigName("config")
		}

		viper.SetEnvPrefix("transcode")
		viper.AutomaticEnv() // read in environment variables that match

		err := viper.ReadInConfig()
		if err != nil && cfgFile != "" {
			log.Err(err)
		}

		// all configs (from file, env and flags) are loaded now,
		// we can set them
		config := transcode.Service.RootConfig
		config.Set()

		if config.Debug {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		} else {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		}

		file := viper.ConfigFileUsed()
		if file != "" {
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Info().Msg("config file reloaded")
				transcode.Service.ConfigReload()
			})

			viper.WatchConfig()

			log.Info().
				Bool("debug", config.Debug).
				Str("config", file).
				Msg("preflight complete with config file")
		} else {
			log.Warn().
				Bool("debug", config.Debug).
				Msg("preflight complete without config file")
		}
	})

	if err := transcode.Service.RootConfig.Init(root); err != nil {
		log.Panic().Err(err).Msg("unable to run root command")
	}
}
