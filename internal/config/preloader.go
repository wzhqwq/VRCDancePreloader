package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) PreloaderMaxCount() interactive.StatefulSetting[int] {
	return m.cache.NewIntSetting(
		"preloader.max-preload",
		func(cfg Config) int {
			return cfg.Preloader.MaxPreload
		},
		func(cfg *Config, count int) error {
			cfg.Preloader.MaxPreload = count
			return nil
		},
	)
}

func (m *Manager) PreloaderEnabledRooms() interactive.StatefulSetting[[]string] {
	return m.cache.NewStringListSetting(
		"preloader.enabled-rooms",
		func(cfg Config) []string {
			return cfg.Preloader.EnabledRooms
		},
		func(cfg *Config, rooms []string) error {
			cfg.Preloader.EnabledRooms = rooms
			return nil
		},
	)
}

//var pypySupportedPlatforms = []string{
//	"PyPyDance",
//	"BiliBili",
//	//"YouTube",
//}
//var wannaSupportedPlatforms = []string{
//	"WannaDance",
//	"BiliBili",
//	//"YouTube",
//}
//
//func checkPreloadConflict() {
//	pypyHasEnabledPlatforms := false
//	wannaHasEnabledPlatforms := false
//
//	for _, platform := range pypySupportedPlatforms {
//		if lo.IndexOf(config.Preload.EnabledPlatforms, platform) != -1 {
//			pypyHasEnabledPlatforms = true
//		}
//	}
//	for _, platform := range wannaSupportedPlatforms {
//		if lo.IndexOf(config.Preload.EnabledPlatforms, platform) != -1 {
//			wannaHasEnabledPlatforms = true
//		}
//	}
//
//	if !pypyHasEnabledPlatforms {
//		if index := lo.IndexOf(config.Preload.EnabledRooms, "PyPyDance"); index != -1 {
//			config.Preload.EnabledRooms = slices.Delete(config.Preload.EnabledRooms, index, index+1)
//		}
//	}
//	if !wannaHasEnabledPlatforms {
//		if index := lo.IndexOf(config.Preload.EnabledRooms, "WannaDance"); index != -1 {
//			config.Preload.EnabledRooms = slices.Delete(config.Preload.EnabledRooms, index, index+1)
//		}
//	}
//}
