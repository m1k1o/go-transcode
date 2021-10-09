package hlsvod

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
