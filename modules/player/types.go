package player

type Config struct {
	Source string
}

func (c Config) withDefaultValues() Config {
	if c.Source == "" {
		c.Source = "index.m3u8"
	}
	return c
}
