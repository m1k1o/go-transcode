package server

import (
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	Bind    string `mapstructure:"bind"`
	Static  string `mapstructure:"static"`
	SSLCert string `mapstructure:"sslcert"`
	SSLKey  string `mapstructure:"sslkey"`
	Proxy   bool   `mapstructure:"proxy"`
	PProf   bool   `mapstructure:"pprof"`
}

func (Config) Init(cmd *cobra.Command) error {
	cmd.PersistentFlags().String("bind", "127.0.0.1:8080", "address/port/socket to serve http")
	if err := viper.BindPFlag("bind", cmd.PersistentFlags().Lookup("bind")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("static", "", "path to client files to serve")
	if err := viper.BindPFlag("static", cmd.PersistentFlags().Lookup("static")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("sslcert", "", "path to the SSL cert")
	if err := viper.BindPFlag("sslcert", cmd.PersistentFlags().Lookup("sslcert")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("sslkey", "", "path to the SSL key")
	if err := viper.BindPFlag("sslkey", cmd.PersistentFlags().Lookup("sslkey")); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("proxy", false, "allow reverse proxies")
	if err := viper.BindPFlag("proxy", cmd.PersistentFlags().Lookup("proxy")); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("pprof", false, "enable pprof endpoint available at /debug/pprof")
	if err := viper.BindPFlag("pprof", cmd.PersistentFlags().Lookup("pprof")); err != nil {
		return err
	}

	return nil
}

func (c *Config) Set() {
	if err := viper.Unmarshal(c); err != nil {
		log.Panic().Msg("unable to unmarshal config structure")
	}
}
