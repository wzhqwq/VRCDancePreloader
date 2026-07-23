package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) CachePath() interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"cache_manager.path",
		func(cfg Config) string {
			return cfg.Cache.Path
		},
		func(cfg *Config, p string) error {
			cfg.Cache.Path = p
			return nil
		},
	)
}

func (m *Manager) MaxVideoCache() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"cache_manager.max-video-cache",
		func(cfg Config) int {
			return cfg.Cache.MaxVideoCache
		},
		func(cfg *Config, sizeInMb int) error {
			cfg.Cache.MaxVideoCache = sizeInMb
			return nil
		},
	)
}

func (m *Manager) VideoFileFormat() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"cache_manager.video-file-format",
		func(cfg Config) int {
			return cfg.Cache.VideoFileFormat
		},
		func(cfg *Config, f int) error {
			cfg.Cache.VideoFileFormat = f
			return nil
		},
	)
}

func (m *Manager) CacheForceExpiration() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"cache_manager.force-expiration",
		func(cfg Config) bool {
			return cfg.Cache.ForceExpiration
		},
		func(cfg *Config, f bool) error {
			cfg.Cache.ForceExpiration = f
			return nil
		},
	)
}

func (m *Manager) CacheKeepFavorites() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"cache_manager.keep-favorites",
		func(cfg Config) bool {
			return cfg.Cache.KeepFavorites
		},
		func(cfg *Config, keep bool) error {
			cfg.Cache.KeepFavorites = keep
			return nil
		},
	)
}
