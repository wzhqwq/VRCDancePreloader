package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
)

type CacheConfig struct {
	Path          string `yaml:"path"`
	MaxCacheSize  int    `yaml:"max-cache-size"`
	KeepFavorites bool   `yaml:"keep-favorites"`
	//RWBufferSize  int    `yaml:"rw-buffer-size"`
	// 0: legacy, 1: continuous, 2: fragmented
	FileFormat int `yaml:"file-format"`

	ForceExpirationCheck bool `yaml:"force-expiration-check"`
}

func GetCacheConfig() *CacheConfig {
	return &config.Cache
}

var defaultCacheConfig = CacheConfig{
	Path:         "./cache",
	MaxCacheSize: 300,
	//RWBufferSize:  1,
	FileFormat: 1,
}

func (cc *CacheConfig) Init() {
	if cc.FileFormat <= 0 {
		logger.WarnLn("We no longer support writing legacy cache files. `cache.file-format` will be replaced with default value")
		cc.FileFormat = 1
		saveAndNotify("cache")
	}

	cache.SetupCache(cc.Path)
	video_cache.SetMaxSize(int64(cc.MaxCacheSize) * 1024 * 1024)
	video_cache.SetKeepFavorites(cc.KeepFavorites)
	entry.SetFileFormat(cc.FileFormat)
	entry.SetForceExpirationCheck(cc.ForceExpirationCheck)
}

func (cc *CacheConfig) UpdateMaxSize(sizeInMb int) {
	cc.MaxCacheSize = sizeInMb
	video_cache.SetMaxSize(int64(sizeInMb) * 1024 * 1024)
	cache.CleanUpCache()
	saveAndNotify("cache")
}

func (cc *CacheConfig) UpdateKeepFavorites(b bool) {
	cc.KeepFavorites = b
	video_cache.SetKeepFavorites(b)
	saveAndNotify("cache")
}

func (cc *CacheConfig) UpdateForceExpirationCheck(b bool) {
	cc.ForceExpirationCheck = b
	entry.SetForceExpirationCheck(b)
	saveAndNotify("cache")
}

func (cc *CacheConfig) UpdateFileFormat(fileFormat int) {
	cc.FileFormat = fileFormat
	entry.SetFileFormat(fileFormat)
	saveAndNotify("cache")
}

func (cc *CacheConfig) UpdatePath(path string) {
	cc.Path = path
	saveAndNotify("cache")
}
