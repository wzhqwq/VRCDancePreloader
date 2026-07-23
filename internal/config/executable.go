package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) ExecutableAutoCheck() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"executable.check-update-on-start",
		func(c Config) bool {
			return c.Executable.CheckUpdateOnStart
		},
		func(c *Config, enable bool) error {
			c.Executable.CheckUpdateOnStart = enable
			return nil
		},
	)
}

func (m *Manager) ExecutableYtDlpPath() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"executable.yt-dlp-path",
		func(c Config) string {
			return c.Executable.YtDlpPath
		},
		func(c *Config, path string) error {
			c.Executable.YtDlpPath = path
			return nil
		},
	)
}

func (m *Manager) ExecutableDenoPath() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"executable.deno-path",
		func(c Config) string {
			return c.Executable.DenoPath
		},
		func(c *Config, path string) error {
			c.Executable.DenoPath = path
			return nil
		},
	)
}
