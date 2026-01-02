package config

import (
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
)

func checkInterceptionConflict() {
	pypyIntercepted := false
	wannaIntercepted := false
	biliIntercepted := false
	for _, site := range config.Hijack.InterceptedSites {
		if constants.IsPyPySite(site) {
			pypyIntercepted = true
		}
		if constants.IsWannaSite(site) {
			wannaIntercepted = true
		}
		if constants.IsBiliSite(site) {
			biliIntercepted = true
		}
	}
	if !pypyIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledPlatforms, "PyPyDance"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			logger.WarnLn("[Config Changed] According to the hijack config, none of the video sources providing PyPyDance videos are intercepted, so the PyPyDance video will not be preloaded.")
			logger.InfoLn("Valid sources for PyPyDance:", strings.Join(constants.AllPyPySites(), ", "))
		}
	}
	if !wannaIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "WannaDance"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			logger.WarnLn("[Config Changed] According to the hijack config, none of the video sources providing WannaDance videos are intercepted, so the WannaDance video will not be preloaded.")
			logger.InfoLn("Valid sources for WannaDance:", strings.Join(constants.AllWannaSites(), ", "))
		}
	}
	if !biliIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "BiliBili"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			logger.WarnLn("[Config Changed] According to the hijack config, none of the video sources providing BiliBili videos are intercepted, so the BiliBili video will not be preloaded.")
			logger.InfoLn("Valid sources for BiliBili:", strings.Join(constants.AllBiliSites(), ", "))
		}
	}
}
