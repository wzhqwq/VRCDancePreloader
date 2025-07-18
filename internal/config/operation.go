package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/hijack"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"log"
)

func (hc *HijackConfig) Init() {
	hc.HijackRunner = NewHijackServerRunner()
	hc.HijackRunner.Run()
	if hc.EnablePWI {
		service.StartPWIServer()
	}
}

func (hc *HijackConfig) Stop() {
	hijack.Stop()
}

func (hc *HijackConfig) UpdatePort(port int) {
	hc.ProxyPort = port
	SaveConfig()
}

func (hc *HijackConfig) startHijack() error {
	return hijack.Start(hc.InterceptedSites, hc.EnableHttps, hc.ProxyPort)
}

func (hc *HijackConfig) UpdateEnableHttps(b bool) {
	hc.EnableHttps = b
	hc.HijackRunner.Run()
	SaveConfig()
}

func (hc *HijackConfig) UpdateSites(sites []string) {
	hc.InterceptedSites = sites
	hc.HijackRunner.Run()
	SaveConfig()
}

func (hc *HijackConfig) UpdateEnablePWI(b bool) {
	hc.EnablePWI = b
	if hc.EnablePWI {
		service.StartPWIServer()
	} else {
		service.StopPWIServer()
	}
}

func (pc *ProxyConfig) Init() {
	//TODO cancel comment after implemented youtube preloading
	pc.ProxyControllers = map[string]*ProxyTester{
		"pypydance-api": NewProxyTester("pypydance-api", pc.Pypy),
		"youtube-video": NewProxyTester("youtube-video", pc.YoutubeVideo),
		"youtube-api":   NewProxyTester("youtube-api", pc.YoutubeApi),
		"youtube-image": NewProxyTester("youtube-image", pc.YoutubeImage),
	}

	requesting.InitPypyClient(pc.Pypy)
	requesting.InitWannaClient("")
	requesting.InitBiliClient("")
	//requesting.InitYoutubeVideoClient(pc.YoutubeVideo)
	requesting.InitYoutubeImageClient(pc.YoutubeImage)
	requesting.InitYoutubeApiClient(pc.YoutubeApi)

	if !skipTest {
		pc.ProxyControllers["pypydance-api"].Test()
	}
	//if config.Youtube.EnableVideo {
	//	if !skipTest {
	//		pc.ProxyControllers["youtube-video"].Test()
	//	}
	//}
	if config.Youtube.EnableThumbnail {
		if !skipTest {
			pc.ProxyControllers["youtube-image"].Test()
		}
	}
	if config.Youtube.EnableApi {
		if !skipTest {
			pc.ProxyControllers["youtube-api"].Test()
		}
	}
}

func (pc *ProxyConfig) Update(item, value string) {
	switch item {
	case "pypydance-api":
		pc.Pypy = value
		requesting.InitPypyClient(value)
	case "youtube-video":
		pc.YoutubeVideo = value
		requesting.InitYoutubeVideoClient(value)
	case "youtube-api":
		pc.YoutubeApi = value
		requesting.InitYoutubeApiClient(value)
	case "youtube-image":
		pc.YoutubeImage = value
		requesting.InitYoutubeImageClient(value)
	default:
		log.Fatalf("Unknown proxy item: %s", item)
	}
	SaveConfig()
}

func (pc *ProxyConfig) Test(item string) (bool, string) {
	switch item {
	case "pypydance-api":
		return requesting.TestPypyClient()
	case "youtube-video":
		return requesting.TestYoutubeVideoClient()
	case "youtube-api":
		return requesting.TestYoutubeApiClient()
	case "youtube-image":
		return requesting.TestYoutubeImageClient()
	default:
		log.Fatalf("Unknown proxy item: %s", item)
		return false, ""
	}
}

func (kc *KeyConfig) Init() {
	if config.Youtube.EnableApi {
		if kc.Youtube == "" {
			third_party_api.YoutubeApiKey = kc.Youtube
		} else {
			log.Println("Youtube API feature is disabled because Youtube API key is missing")
			config.Youtube.UpdateEnableApi(false)
		}
	}
}

func (yc *YoutubeConfig) Init() {
	third_party_api.EnableYoutubeApi = yc.EnableApi
	third_party_api.EnableYoutubeThumbnail = yc.EnableThumbnail
}

func (yc *YoutubeConfig) UpdateEnableApi(enabled bool) {
	yc.EnableApi = enabled
	third_party_api.EnableYoutubeApi = enabled
	SaveConfig()
}

func (yc *YoutubeConfig) UpdateEnableThumbnail(enabled bool) {
	yc.EnableThumbnail = enabled
	third_party_api.EnableYoutubeThumbnail = enabled
	SaveConfig()
}

func (pc *PreloadConfig) Init() {
	playlist.Init(pc.MaxPreload)
	playlist.SetEnabledRooms(pc.EnabledRooms)
	playlist.SetEnabledPlatforms(pc.EnabledPlatforms)
}

func (pc *PreloadConfig) UpdateMaxPreload(max int) {
	pc.MaxPreload = max
	playlist.SetMaxPreload(max)
	SaveConfig()
}

func (dc *DownloadConfig) Init() {
	download.InitDownloadManager(dc.MaxDownload)
}

func (dc *DownloadConfig) UpdateMaxDownload(max int) {
	dc.MaxDownload = max
	download.SetMaxParallel(max)
	SaveConfig()
}

func (cc *CacheConfig) Init() {
	cache.SetupCache(cc.Path)
	cache.SetMaxSize(int64(cc.MaxCacheSize) * 1024 * 1024)
	cache.SetKeepFavorites(cc.KeepFavorites)
}

func (cc *CacheConfig) UpdateMaxSize(sizeInMb int) {
	cc.MaxCacheSize = sizeInMb
	cache.SetMaxSize(int64(sizeInMb) * 1024 * 1024)
	SaveConfig()
}

func (cc *CacheConfig) UpdateKeepFavorites(b bool) {
	cc.KeepFavorites = b
	cache.SetKeepFavorites(b)
	SaveConfig()
}

func (dc *DbConfig) Init() error {
	err := persistence.InitDB(dc.Path)
	if err != nil {
		return err
	}
	return nil
}
