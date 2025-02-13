package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"log"
)

func (pc *ProxyConfig) Init() {
	pc.ProxyControllers = map[string]*ProxyController{
		"pypydance-api": NewProxyController("pypydance-api", pc.Pypy),
		"youtube-video": NewProxyController("youtube-video", pc.YoutubeVideo),
		"youtube-api":   NewProxyController("youtube-api", pc.YoutubeApi),
		"youtube-image": NewProxyController("youtube-image", pc.YoutubeImage),
	}

	requesting.InitPypyClient(pc.Pypy)
	requesting.InitYoutubeVideoClient(pc.YoutubeVideo)
	requesting.InitYoutubeImageClient(pc.YoutubeImage)
	requesting.InitYoutubeApiClient(pc.YoutubeApi)

	if !skipTest {
		pc.ProxyControllers["pypydance-api"].Test()
	}
	if config.Youtube.EnableVideo {
		if !skipTest {
			pc.ProxyControllers["youtube-video"].Test()
		}
	}
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
		if kc.Youtube != "" {
			third_party_api.SetYoutubeApiKey(kc.Youtube)
		} else {
			log.Fatalf("Youtube API key must be set when Youtube API feature is enabled")
		}
	}
}

func (pc *PreloadConfig) Init() {
	playlist.Init(pc.MaxPreload)
}

func (dc *DownloadConfig) Init() {
	download.InitDownloadManager(dc.MaxDownload)
}

func (cc *CacheConfig) Init() {
	cache.SetupCache(cc.Path, cc.MaxCacheSize)
}
