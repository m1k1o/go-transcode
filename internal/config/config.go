package config

import (
	"fmt"
	"os"
	"path"

	"github.com/m1k1o/go-transcode/internal/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type VideoProfile struct {
	Width   int `mapstructure:"width"`
	Height  int `mapstructure:"height"`
	Bitrate int `mapstructure:"bitrate"` // in kilobytes
}

type AudioProfile struct {
	Bitrate int `mapstructure:"bitrate"` // in kilobytes
}

type VOD struct {
	MediaDir       string                  `mapstructure:"media-dir"`
	TranscodeDir   string                  `mapstructure:"transcode-dir"`
	VideoProfiles  map[string]VideoProfile `mapstructure:"video-profiles"`
	VideoKeyframes bool                    `mapstructure:"video-keyframes"`
	AudioProfile   AudioProfile            `mapstructure:"audio-profile"`
	Cache          bool                    `mapstructure:"cache"`
	CacheDir       string                  `mapstructure:"cache-dir"`
	FFmpegBinary   string                  `mapstructure:"ffmpeg-binary"`
	FFprobeBinary  string                  `mapstructure:"ffprobe-binary"`
}

type Server struct {
	Server server.Config

	BaseDir  string            `yaml:"basedir,omitempty"`
	Streams  map[string]string `yaml:"streams"`
	Profiles string            `yaml:"profiles,omitempty"`

	Vod      VOD
	HlsProxy map[string]string
}

func (s *Server) Init(cmd *cobra.Command) error {
	// TODO: Scope
	if err := s.Server.Init(cmd); err != nil {
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

	return nil
}

func (s *Server) Set() {
	s.Server.Set()

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

	//
	// VOD
	//
	if err := viper.UnmarshalKey("vod", &s.Vod); err != nil {
		panic(err)
	}

	// defaults

	if s.Vod.TranscodeDir == "" {
		var err error
		s.Vod.TranscodeDir, err = os.MkdirTemp(os.TempDir(), "go-transcode-vod")
		if err != nil {
			panic(err)
		}
	} else {
		err := os.MkdirAll(s.Vod.TranscodeDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	if len(s.Vod.VideoProfiles) == 0 {
		panic("specify at least one VOD video profile")
	}

	if s.Vod.Cache && s.Vod.CacheDir != "" {
		err := os.MkdirAll(s.Vod.CacheDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	if s.Vod.FFmpegBinary == "" {
		s.Vod.FFmpegBinary = "ffmpeg"
	}

	if s.Vod.FFprobeBinary == "" {
		s.Vod.FFprobeBinary = "ffprobe"
	}

	//
	// HLS PROXY
	//
	s.HlsProxy = viper.GetStringMapString("hls-proxy")
}

func (s *Server) AbsPath(elem ...string) string {
	// prepend base path
	elem = append([]string{s.BaseDir}, elem...)
	return path.Join(elem...)
}
