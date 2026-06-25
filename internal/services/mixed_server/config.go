package mixed_server

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
)

type Config struct {
	Port int `yaml:"port"`

	InterceptedSites []string `yaml:"intercepted-sites"`
	EnableHttpsProxy bool     `yaml:"enable-https-proxy"`
}

func DefaultConfig() Config {
	return Config{
		Port:             7653,
		InterceptedSites: constants.CopyAllSites(),
	}
}

func (c Config) Validate() error {
	if c.Port < 0 || c.Port > 65535 {
		return errors.New("invalid port")
	}
	return nil
}
