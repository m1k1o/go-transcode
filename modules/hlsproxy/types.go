package hlsproxy

import "github.com/m1k1o/go-transcode/pkg/hlsproxy"

type Config struct {
	hlsproxy.Config

	// overwritten properties
	PlaylistBaseUrl    string `mapstructure:"-"`
	PlaylistPathPrefix string `mapstructure:"-"`
	SegmentBaseUrl     string `mapstructure:"-"`
	SegmentPathPrefix  string `mapstructure:"-"`

	Sources map[string]string
}

func (c Config) withDefaultValues() Config {
	return c
}
