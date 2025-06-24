package config

import (
	"github.com/samber/lo"
	"log"
	"slices"
	"strings"
)

var pypySites = []string{
	"jd.pypy.moe",
}
var wannaSites = []string{
	"api.udon.dance",
	"api.wannadance.online",
}
var biliSites = []string{
	"www.bilibili.com",
	"b23.tv",
	"api.xin.moe",
}
var allSites = []string{
	// PyPyDance
	"jd.pypy.moe",
	// WannaDance
	"api.udon.dance",
	"api.wannadance.online",
	// BiliBili
	"www.bilibili.com",
	"b23.tv",
	"api.xin.moe",
}

func checkInterceptionConflict() {
	pypyIntercepted := false
	wannaIntercepted := false
	biliIntercepted := false
	for _, site := range config.Hijack.InterceptedSites {
		if lo.IndexOf(pypySites, site) != -1 {
			pypyIntercepted = true
		}
		if lo.IndexOf(wannaSites, site) != -1 {
			wannaIntercepted = true
		}
		if lo.IndexOf(biliSites, site) != -1 {
			biliIntercepted = true
		}
	}
	if !pypyIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledPlatforms, "PyPyDance"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			log.Println("[Config Changed] According to the hijack config, none of the video sources providing PyPyDance videos are intercepted, so the PyPyDance video will not be preloaded.")
			log.Println("Valid sources for PyPyDance: " + strings.Join(pypySites, ", "))
		}
	}
	if !wannaIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "WannaDance"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			log.Println("[Config Changed] According to the hijack config, none of the video sources providing WannaDance videos are intercepted, so the WannaDance video will not be preloaded.")
			log.Println("Valid sources for WannaDance: " + strings.Join(wannaSites, ", "))
		}
	}
	if !biliIntercepted {
		if index := lo.IndexOf(config.Preload.EnabledRooms, "BiliBili"); index != -1 {
			config.Preload.EnabledPlatforms = slices.Delete(config.Preload.EnabledPlatforms, index, index+1)
			log.Println("[Config Changed] According to the hijack config, none of the video sources providing BiliBili videos are intercepted, so the BiliBili video will not be preloaded.")
			log.Println("Valid sources for BiliBili: " + strings.Join(biliSites, ", "))
		}
	}
}
