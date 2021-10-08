package hlsvod

type Config struct {
	MediaPath    string // Transcoded video input.
	TranscodeDir string // Temporary directory to store transcoded elements.

	FFmpegBinary  string
	FFprobeBinary string
}
