package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"log"
)

func (pc *ProxyConfig) Init() {
	requesting.InitPypyClient(pc.Pypy)
	if config.Youtube.EnableVideo {
		requesting.InitYoutubeVideoClient(pc.YoutubeVideo)
	}
	if config.Youtube.EnableThumbnail {
		requesting.InitYoutubeImageClient(pc.YoutubeImage)
	}
	if config.Youtube.EnableApi {
		requesting.InitYoutubeApiClient(pc.YoutubeApi)
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
