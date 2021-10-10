package config

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Root struct {
	Debug   bool
	PProf   bool
	CfgFile string
}

func (Root) Init(cmd *cobra.Command) error {
	cmd.PersistentFlags().BoolP("debug", "d", false, "enable debug mode")
	if err := viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug")); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("pprof", false, "enable pprof endpoint available at /debug/pprof")
	if err := viper.BindPFlag("pprof", cmd.PersistentFlags().Lookup("pprof")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("config", "", "configuration file path")
	if err := viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config")); err != nil {
		return err
	}

	return nil
}

func (s *Root) Set() {
	s.Debug = viper.GetBool("debug")
	s.PProf = viper.GetBool("pprof")
	s.CfgFile = viper.GetString("config")
}

type Server struct {
	Cert   string
	Key    string
	Bind   string
	Static string
	Proxy  bool

	BaseDir  string            `yaml:"basedir,omitempty"`
	Streams  map[string]string `yaml:"streams"`
	Profiles string            `yaml:"profiles,omitempty"`

	VodDir string
}

func (Server) Init(cmd *cobra.Command) error {
	cmd.PersistentFlags().String("bind", "127.0.0.1:8080", "address/port/socket to serve neko")
	if err := viper.BindPFlag("bind", cmd.PersistentFlags().Lookup("bind")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("cert", "", "path to the SSL cert used to secure the neko server")
	if err := viper.BindPFlag("cert", cmd.PersistentFlags().Lookup("cert")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("key", "", "path to the SSL key used to secure the neko server")
	if err := viper.BindPFlag("key", cmd.PersistentFlags().Lookup("key")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("static", "", "path to neko client files to serve")
	if err := viper.BindPFlag("static", cmd.PersistentFlags().Lookup("static")); err != nil {
		return err
	}

	cmd.PersistentFlags().Bool("proxy", false, "allow reverse proxies")
	if err := viper.BindPFlag("proxy", cmd.PersistentFlags().Lookup("proxy")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("basedir", "", "base directory for assets and profiles")
	if err := viper.BindPFlag("basedir", cmd.PersistentFlags().Lookup("basedir")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("profiles", "", "hardware encoding profiles to load for ffmpeg (default, nvidia)")
	if err := viper.BindPFlag("profiles", cmd.PersistentFlags().Lookup("profiles")); err != nil {
		return err
	}

	cmd.PersistentFlags().String("voddir", "", "vod dir")
	if err := viper.BindPFlag("voddir", cmd.PersistentFlags().Lookup("voddir")); err != nil {
		return err
	}

	return nil
}

func (s *Server) Set() {
	s.Cert = viper.GetString("cert")
	s.Key = viper.GetString("key")
	s.Bind = viper.GetString("bind")
	s.Static = viper.GetString("static")
	s.Proxy = viper.GetBool("proxy")

	s.BaseDir = viper.GetString("basedir")
	if s.BaseDir == "" {
		if _, err := os.Stat("/etc/transcode"); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			s.BaseDir = cwd
		} else {
			s.BaseDir = "/etc/transcode"
		}
	}

	s.Profiles = viper.GetString("profiles")
	if s.Profiles == "" {
		// TODO: issue #5
		s.Profiles = fmt.Sprintf("%s/profiles", s.BaseDir)
	}
	s.Streams = viper.GetStringMapString("streams")

	s.VodDir = viper.GetString("voddir")
}

func (s *Server) AbsPath(elem ...string) string {
	// prepend base path
	elem = append([]string{s.BaseDir}, elem...)
	return path.Join(elem...)
}
