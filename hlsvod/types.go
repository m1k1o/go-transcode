package hlsvod

import (
	"context"
	"net/http"
)

type Config struct {
	MediaPath     string // Transcoded video input.
	TranscodeDir  string // Temporary directory to store transcoded elements.
	SegmentPrefix string

	VideoProfile   *VideoProfile
	VideoKeyframes bool
	AudioProfile   *AudioProfile

	// HLS-VOD segment parameters (override defaults from server)
	SegmentLength    float64
	SegmentOffset    float64
	SegmentBufferMin int
	SegmentBufferMax int

	Cache    bool
	CacheDir string // If not empty, cache will folder will be used instead of media path

	FFmpegBinary  string
	FFprobeBinary string
}

type Manager interface {
	Start() error
	Stop()
	Preload(ctx context.Context) (*ProbeMediaData, error)

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
