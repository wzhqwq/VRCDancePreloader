package downloader

type Config struct {
	MaxDownload int `yaml:"max-parallel"`

	// pypydance, wannadance, dudu-fitdance
	UseYoutubeFallback []string `yaml:"use-youtube-fallback"`

	ThrottledFallback     bool `yaml:"throttled-fallback"`
	LowSpeedFallback      bool `yaml:"low-speed-fallback"`
	HighFramerateFallback bool `yaml:"high-framerate-fallback"`
}

func (c Config) Validate(_ string) error {
	return nil
}

func DefaultConfig() Config {
	return Config{
		MaxDownload: 1,
	}
}
