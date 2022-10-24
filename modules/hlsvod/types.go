package hlsvod

import "github.com/m1k1o/go-transcode/pkg/hlsvod"

type Config struct {
	hlsvod.Config

	// overwritten properties
	MediaPath         string               `mapstructure:"-"`
	SegmentNamePrefix string               `mapstructure:"-"`
	VideoProfile      *hlsvod.VideoProfile `mapstructure:"-"`

	// modified properties
	MediaBasePath string
	TranscodeDir  string

	VideoProfiles      map[string]hlsvod.VideoProfile
	MasterPlaylistName string
}

func (c Config) withDefaultValues() Config {
	if c.MasterPlaylistName == "" {
		c.MasterPlaylistName = "index.m3u8"
	}
	return c
}
