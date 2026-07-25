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

func (m *Manager) ThirdPartyYouTubeAllowedResources() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"third_parties.youtube-allow-resources",
		func(c Config) []string {
			return c.ThirdParties.YoutubeResources
		},
		func(c *Config, resources []string) error {
			c.ThirdParties.YoutubeResources = resources
			return nil
		},
	)
}

func (m *Manager) ThirdPartyBiliBiliAllowedResources() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"third_parties.bilibili-allow-resources",
		func(c Config) []string {
			return c.ThirdParties.BiliBiliResources
		},
		func(c *Config, resources []string) error {
			c.ThirdParties.BiliBiliResources = resources
			return nil
		},
	)
}

func (m *Manager) ThirdPartyPyPyDanceAllowedResources() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"third_parties.pypydance-allow-resources",
		func(c Config) []string {
			return c.ThirdParties.PyPyDanceResources
		},
		func(c *Config, resources []string) error {
			c.ThirdParties.PyPyDanceResources = resources
			return nil
		},
	)
}

func (m *Manager) ThirdPartyWannaDanceAllowedResources() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"third_parties.wannadance-allow-resources",
		func(c Config) []string {
			return c.ThirdParties.WannaDanceResources
		},
		func(c *Config, resources []string) error {
			c.ThirdParties.WannaDanceResources = resources
			return nil
		},
	)
}

func (m *Manager) ThirdPartyDuDuFitDanceAllowedResources() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"third_parties.dudu-fitdance-allow-resources",
		func(c Config) []string {
			return c.ThirdParties.DuDuFitDanceResources
		},
		func(c *Config, resources []string) error {
			c.ThirdParties.DuDuFitDanceResources = resources
			return nil
		},
	)
}
