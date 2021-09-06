package api

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type YamlConf struct {
	Streams map[string]string `yaml:"streams"`
}

func loadConf(path string) (*YamlConf, error) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	conf := &YamlConf{}
	err = yaml.Unmarshal(yamlFile, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
