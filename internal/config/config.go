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

	if conf.BaseDir == "" {
		cwd, _ := os.Getwd()
		conf.BaseDir = cwd
	}

	return conf, nil
}
