package config

import (
	"github.com/samber/lo"
	"slices"
)

var pypySupportedPlatforms = []string{
	"PyPyDance",
	"BiliBili",
	//"YouTube",
}
var wannaSupportedPlatforms = []string{
	"WannaDance",
	"BiliBili",
	//"YouTube",
}

func checkPreloadConflict() {
	pypyHasEnabledPlatforms := false
	wannaHasEnabledPlatforms := false

	for _, platform := range pypySupportedPlatforms {
		if lo.IndexOf(config.Preload.EnabledPlatforms, platform) != -1 {
			pypyHasEnabledPlatforms = true
		}
	}
	for _, platform := range wannaSupportedPlatforms {
		if lo.IndexOf(config.Preload.EnabledPlatforms, platform) != -1 {
			wannaHasEnabledPlatforms = true
		}
	}

	if !pypyHasEnabledPlatforms {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "PyPyDance"); index != -1 {
			config.Preload.EnabledRooms = slices.Delete(config.Preload.EnabledRooms, index, index+1)
		}
	}
	if !wannaHasEnabledPlatforms {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "WannaDance"); index != -1 {
			config.Preload.EnabledRooms = slices.Delete(config.Preload.EnabledRooms, index, index+1)
		}
	}
}
