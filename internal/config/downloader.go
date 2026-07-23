package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) DownloaderMaxParallel() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"downloader.max-parallel",
		func(cfg Config) int {
			return cfg.Downloader.MaxDownload
		},
		func(cfg *Config, count int) error {
			cfg.Downloader.MaxDownload = count
			return nil
		},
	)
}

func (m *Manager) DownloaderYouTubeFallback() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"downloader.use-youtube-fallback",
		func(cfg Config) []string {
			return cfg.Downloader.UseYoutubeFallback
		},
		func(cfg *Config, rooms []string) error {
			cfg.Downloader.UseYoutubeFallback = rooms
			return nil
		},
	)
}

func (m *Manager) DownloaderThrottledFallback() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"downloader.throttled-fallback",
		func(cfg Config) bool {
			return cfg.Downloader.ThrottledFallback
		},
		func(cfg *Config, use bool) error {
			cfg.Downloader.ThrottledFallback = use
			return nil
		},
	)
}

func (m *Manager) DownloaderLowSpeedFallback() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"downloader.low-speed-fallback",
		func(cfg Config) bool {
			return cfg.Downloader.LowSpeedFallback
		},
		func(cfg *Config, use bool) error {
			cfg.Downloader.LowSpeedFallback = use
			return nil
		},
	)
}

func (m *Manager) DownloaderHighFramerateFallback() interactive.StatefulSetting[bool] {
	return m.cache.NewBoolSetting(
		"downloader.high-framerate-fallback",
		func(cfg Config) bool {
			return cfg.Downloader.HighFramerateFallback
		},
		func(cfg *Config, use bool) error {
			cfg.Downloader.HighFramerateFallback = use
			return nil
		},
	)
}
