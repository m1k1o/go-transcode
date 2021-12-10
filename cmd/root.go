package cmd

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Default configuration path
const defCfgPath = "/etc/transcode/"

// ENV prefix for configuration
const envPrefix = "TRANSCODE"

var rootCmd = &cobra.Command{
	Use:     "transcode",
	Short:   "Transcode server CLI.",
	Long:    `Transcode HTTP on-demand transcoding API.`,
	Version: "1.0.0",
}

var onConfigLoad []func()

func init() {
	var cfgFile string
	var logConfig logConfig

	cobra.OnInitialize(func() {
		initConfiguration(cfgFile, defCfgPath, envPrefix)
		logConfig.Set()
		initLogging(logConfig)

		// display used configuration file
		file := viper.ConfigFileUsed()
		if file != "" {
			viper.OnConfigChange(func(e fsnotify.Event) {
				log.Info().Msg("config file reloaded")

				// call load config
				for _, loadConfig := range onConfigLoad {
					loadConfig()
				}
			})

			viper.WatchConfig()

			log.Info().Str("config", file).Msg("preflight complete with config file")
		} else {
			log.Warn().Msg("preflight complete without config file")
		}

		// call load config
		for _, loadConfig := range onConfigLoad {
			loadConfig()
		}
	})

	// config file
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path")
	_ = viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	// log configuration
	_ = logConfig.Init(rootCmd)
}

type Config interface {
	Init(cmd *cobra.Command) error
	Set()
}

func Execute() error {
	return rootCmd.Execute()
}

//
// Configuration initialization
//

func initConfiguration(cfgFile string, defCfgPath string, envPrefix string) {
	// use configuration file if provided
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// confguratino file name
		viper.SetConfigName("config")

		// search for configuration file
		if runtime.GOOS == "linux" && defCfgPath != "" {
			viper.AddConfigPath(defCfgPath)
		}

		// seatch for configuration file in ./
		viper.AddConfigPath(".")
	}

	if envPrefix != "" {
		// env prefix is uppercase progname
		viper.SetEnvPrefix(envPrefix)

		// replace . and - with _
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

		// read in environment variables that match
		viper.AutomaticEnv()
	}

	// read config file
	err := viper.ReadInConfig()
	if err != nil && cfgFile != "" {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}

//
// Logging initialization
//

type logConfig struct {
	// Set log level
	Level string `yaml:"level"`
	// Enable console logging
	Console bool `yaml:"console"`
	// Enable file logging and specify its path
	File string `yaml:"file"`
	// MaxAge the max age in days to keep a logfile
	MaxAge int `yaml:"maxage"`
	// MaxSize the max size in MB of the logfile before it's rolled
	MaxSize int `yaml:"maxsize"`
	// MaxBackups the max number of rolled files to keep
	MaxBackups int `yaml:"maxbackups"`
}

func (logConfig) Init(cmd *cobra.Command) error {
	cmd.PersistentFlags().String("log.level", "", "Set log level")
	if err := viper.BindPFlag("log.level", cmd.PersistentFlags().Lookup("log.level")); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("log.console", true, "Enable console logging")
	if err := viper.BindPFlag("log.console", cmd.PersistentFlags().Lookup("log.console")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("log.file", "", "Enable file logging and specify its path")
	if err := viper.BindPFlag("log.file", cmd.PersistentFlags().Lookup("log.file")); err != nil {
		return err
	}

	cmd.PersistentFlags().Int("log.maxage", 0, "MaxAge the max age in days to keep a logfile")
	if err := viper.BindPFlag("log.maxage", cmd.PersistentFlags().Lookup("log.maxage")); err != nil {
		return err
	}

	cmd.PersistentFlags().Int("log.maxsize", 100, "MaxSize the max size in MB of the logfile before it's rolled")
	if err := viper.BindPFlag("log.maxsize", cmd.PersistentFlags().Lookup("log.maxsize")); err != nil {
		return err
	}

	cmd.PersistentFlags().Int("log.maxbackups", 0, "MaxBackups the max number of rolled files to keep")
	if err := viper.BindPFlag("log.maxbackups", cmd.PersistentFlags().Lookup("log.maxbackups")); err != nil {
		return err
	}

	return nil
}

func (c *logConfig) Set() {
	c.Level = viper.GetString("log.level")
	c.Console = viper.GetBool("log.console")
	c.File = viper.GetString("log.file")
	c.MaxAge = viper.GetInt("log.maxage")
	c.MaxSize = viper.GetInt("log.maxsize")
	c.MaxBackups = viper.GetInt("log.maxbackups")
}

func initLogging(config logConfig) {
	var writers []io.Writer

	if config.Console {
		writers = append(writers, zerolog.ConsoleWriter{
			Out: os.Stderr,
		})
	}

	if config.File != "" {
		logger := &lumberjack.Logger{
			Filename:   config.File,
			MaxAge:     config.MaxAge,     // days
			MaxSize:    config.MaxSize,    // megabytes
			MaxBackups: config.MaxBackups, // files
		}

		// rotate in response to SIGHUP
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGHUP)

		go func() {
			for {
				<-c
				logger.Rotate()
			}
		}()

		writers = append(writers, logger)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(io.MultiWriter(writers...))

	if config.Level == "" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log.Info().Msg("using default log level")
	} else {
		// set custom log level
		level, err := zerolog.ParseLevel(config.Level)
		if err != nil {
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			log.Warn().Str("log-level", config.Level).Msg("unknown log level")
		} else {
			zerolog.SetGlobalLevel(level)
		}
	}

	log.Info().
		Bool("console", config.Console).
		Str("file", config.File).
		Int("maxage", config.MaxAge).
		Int("maxsize", config.MaxSize).
		Int("maxbackups", config.MaxBackups).
		Msg("logging configured")
}
