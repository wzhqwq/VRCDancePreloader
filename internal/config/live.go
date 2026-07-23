package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) LiveServerPort() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"live_server.port",
		func(cfg Config) int {
			return cfg.Live.Port
		},
		func(cfg *Config, port int) error {
			cfg.Live.Port = port
			return nil
		},
	)
}

func (m *Manager) LiveServerEnabled() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"live_server.enabled",
		func(cfg Config) bool {
			return cfg.Live.Enabled
		},
		func(cfg *Config, enabled bool) error {
			cfg.Live.Enabled = enabled
			return nil
		},
	)
}
