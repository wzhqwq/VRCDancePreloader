package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) SecretYoutubeAPIKey() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"secrets.youtube-api-key",
		func(c Config) string {
			return c.Secrets.YoutubeApiKey
		},
		func(c *Config, key string) error {
			c.Secrets.YoutubeApiKey = key
			return nil
		},
	)
}
