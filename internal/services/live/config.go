package live

import (
	"errors"
)

type Config struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func DefaultConfig() Config {
	return Config{
		Port: 7652,
	}
}

func (c Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return errors.New("invalid port")
	}
	return nil
}
