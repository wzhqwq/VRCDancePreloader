package downloader

import "fmt"

type Config struct {
	MaxDownload int `yaml:"max-parallel"`
}

func (c Config) Validate(_ string) error {
	if c.MaxDownload < 0 {
		return fmt.Errorf("invalid count: %d", c.MaxDownload)
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		MaxDownload: 1,
	}
}
