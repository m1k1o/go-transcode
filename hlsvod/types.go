package hlsvod

import "net/http"

type Config struct {
	MediaPath     string // Transcoded video input.
	TranscodeDir  string // Temporary directory to store transcoded elements.
	SegmentPrefix string

	VideoProfile *VideoProfile
	AudioProfile *AudioProfile

	Cache    bool
	CacheDir string // If not empty, cache will folder will be used instead of media path

	FFmpegBinary  string
	FFprobeBinary string
}

type Manager interface {
	Start() error
	Stop()
	Cleanup()

	ServePlaylist(w http.ResponseWriter, r *http.Request)
	ServeMedia(w http.ResponseWriter, r *http.Request)
}
