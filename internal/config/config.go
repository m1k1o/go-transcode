package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
	"github.com/spf13/cobra"
)

type Config interface {
	Init(cmd *cobra.Command) error
	Set()
}

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
