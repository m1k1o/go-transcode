package hlsvod

import (
	"context"
	"net/http"
	"time"
)

type Config struct {
	MediaPath     string // transcoded video input.
	TranscodeDir  string // temporary directory to store transcoded elements.
	SegmentPrefix string

	VideoProfile   *VideoProfile
	VideoKeyframes bool
	AudioProfile   *AudioProfile

	Cache    bool
	CacheDir string // if not empty, cache will folder will be used instead of media path

	FFmpegBinary  string
	FFprobeBinary string

	ReadyTimeout     time.Duration // how long can it take for transcode to be ready
	TranscodeTimeout time.Duration // how long can it take for transcode to be ready

	SegmentLength    float64
	SegmentOffset    float64 // maximim segment length deviation
	SegmentBufferMin int     // minimum segments available after playing head
	SegmentBufferMax int     // maximum segments to be transcoded at once
}

func (c Config) withDefaultValues() Config {
	if c.FFmpegBinary == "" {
		c.FFmpegBinary = "ffmpeg"
	}
	if c.FFprobeBinary == "" {
		c.FFprobeBinary = "ffprobe"
	}
	if c.ReadyTimeout == 0 {
		c.ReadyTimeout = 80 * time.Second
	}
	if c.TranscodeTimeout == 0 {
		c.TranscodeTimeout = 10 * time.Second
	}
	if c.SegmentLength == 0 {
		c.SegmentLength = 3.50
	}
	if c.SegmentOffset == 0 {
		c.SegmentOffset = 1.25
	}
	if c.SegmentBufferMin == 0 {
		c.SegmentBufferMin = 3
	}
	if c.SegmentBufferMax == 0 {
		c.SegmentBufferMax = 5
	}
	return c
}

type Manager interface {
	Start() error
	Stop()
	Preload(ctx context.Context) (*ProbeMediaData, error)

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
