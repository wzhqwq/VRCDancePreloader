package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) ThirdPartyYoutubeMode() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"third_parties.youtube-mode",
		func(c Config) string {
			return c.ThirdParties.YoutubeMode
		},
		func(c *Config, mode string) error {
			c.ThirdParties.YoutubeMode = mode
			return nil
		},
	)
}

func (m *Manager) ThirdPartyBiliBiliMode() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"third_parties.bilibili-mode",
		func(c Config) string {
			return c.ThirdParties.BiliBiliMode
		},
		func(c *Config, mode string) error {
			c.ThirdParties.BiliBiliMode = mode
			return nil
		},
	)
}
