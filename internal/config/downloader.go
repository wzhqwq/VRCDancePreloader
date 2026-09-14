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
