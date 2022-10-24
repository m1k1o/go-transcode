package httpstream

type Config struct {
	Sources      map[string]string
	ProfilesPath string
	UseBufCopy   bool
}

func (c Config) withDefaultValues() Config {
	return c
}
