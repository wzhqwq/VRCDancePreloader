package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) MixedServerPort() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"main_server.port",
		func(cfg Config) int {
			return cfg.Server.Port
		},
		func(cfg *Config, port int) error {
			cfg.Server.Port = port
			return nil
		},
	)
}

func (m *Manager) MixedServerEnableHttps() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"main_server.enable-https-proxy",
		func(cfg Config) bool {
			return cfg.Server.EnableHttpsProxy
		},
		func(cfg *Config, enableHttps bool) error {
			cfg.Server.EnableHttpsProxy = enableHttps
			return nil
		},
	)
}

func (m *Manager) MixedServerInterceptedSites() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"main_server.intercepted-sites",
		func(cfg Config) []string {
			return cfg.Server.InterceptedSites
		},
		func(cfg *Config, sites []string) error {
			cfg.Server.InterceptedSites = sites
			return nil
		},
	)
}
