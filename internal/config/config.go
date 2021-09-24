package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type YamlConf struct {
	BaseDir string `yaml:"basedir",omitempty`
	Streams map[string]string `yaml:"streams"`
}

func LoadConf(path string) (*YamlConf, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &YamlConf{}
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		return nil, err
	}

	// If basedir is not explicit in config, try /etc/go-transcode/,
	// fallback to the current working directory
	if conf.BaseDir == "" {
		if _, err := os.Stat("/etc/go-transcode"); os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			conf.BaseDir = cwd
		} else {
			conf.BaseDir = "/etc/go-transcode"
		}
	}

	return conf, nil
}

type Root struct {
	Debug   bool
	CfgFile string
}

func (Root) Init(cmd *cobra.Command) error {
	cmd.PersistentFlags().BoolP("debug", "d", false, "enable debug mode")
	if err := viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug")); err != nil {
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
	s.CfgFile = viper.GetString("config")
}

type Server struct {
	Cert   string
	Key    string
	Bind   string
	Static string
	Proxy  bool
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

	return nil
}

func (s *Server) Set() {
	s.Cert = viper.GetString("cert")
	s.Key = viper.GetString("key")
	s.Bind = viper.GetString("bind")
	s.Static = viper.GetString("static")
	s.Proxy = viper.GetBool("proxy")
}
