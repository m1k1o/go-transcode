package hlslive

import "github.com/m1k1o/go-transcode/pkg/hlslive"

type Config struct {
	hlslive.Config

	Sources      map[string]string
	ProfilesPath string
	PlaylistName string
}

func (c Config) withDefaultValues() Config {
	if c.PlaylistName == "" {
		c.PlaylistName = "index.m3u8"
	}
	return c
}
