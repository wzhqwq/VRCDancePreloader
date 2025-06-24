package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"sync"
)

type KeyConfig struct {
	Youtube string `yaml:"youtube-api"`
}
type ProxyConfig struct {
	Pypy         string `yaml:"pypydance-api"`
	YoutubeVideo string `yaml:"youtube-video"`
	YoutubeApi   string `yaml:"youtube-api"`
	YoutubeImage string `yaml:"youtube-image"`

	ProxyControllers map[string]*ProxyController `yaml:"-"`
}
type YoutubeConfig struct {
	EnableApi       bool `yaml:"enable-youtube-api"`
	EnableThumbnail bool `yaml:"enable-youtube-thumbnail"`
}
type PreloadConfig struct {
	EnabledRooms     []string `yaml:"enabled-rooms"`
	EnabledPlatforms []string `yaml:"enabled-platforms"`
	MaxPreload       int      `yaml:"max-preload-count"`
}
type HijackConfig struct {
	ProxyPort        int      `yaml:"proxy-port"`
	InterceptedSites []string `yaml:"intercepted-sites"`
	EnableHttps      bool     `yaml:"enable-https"`
	EnablePWI        bool     `yaml:"enable-pwi"`
}
type DownloadConfig struct {
	MaxDownload int `yaml:"max-parallel-download-count"`
}
type CacheConfig struct {
	Path          string `yaml:"path"`
	MaxCacheSize  int    `yaml:"max-cache-size"`
	KeepFavorites bool   `yaml:"keep-favorites"`
}
type DbConfig struct {
	Path string `yaml:"path"`
}

var config struct {
	Version  string         `yaml:"version"`
	Hijack   HijackConfig   `yaml:"hijack"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Keys     KeyConfig      `yaml:"keys"`
	Youtube  YoutubeConfig  `yaml:"youtube"`
	Preload  PreloadConfig  `yaml:"preload"`
	Download DownloadConfig `yaml:"download"`
	Cache    CacheConfig    `yaml:"cache"`
	Db       DbConfig       `yaml:"db"`
}

func FillDefaultSetting() {
	config.Version = "2.2"
	config.Hijack = HijackConfig{
		ProxyPort:        7653,
		InterceptedSites: make([]string, len(allSites)),
		EnableHttps:      true,
		EnablePWI:        false,
	}
	copy(config.Hijack.InterceptedSites, allSites)
	config.Proxy = ProxyConfig{
		Pypy:         "",
		YoutubeVideo: "",
		YoutubeApi:   "",
		YoutubeImage: "",
	}
	config.Keys = KeyConfig{
		Youtube: "",
	}
	config.Youtube = YoutubeConfig{
		EnableApi:       false,
		EnableThumbnail: false,
	}
	config.Preload = PreloadConfig{
		EnabledRooms: []string{
			"PyPyDance",
			"WannaDance",
		},
		EnabledPlatforms: []string{
			"PyPyDance",
			"WannaDance",
			"BiliBili",
			//"YouTube",
		},
		MaxPreload: 4,
	}
	config.Download = DownloadConfig{
		MaxDownload: 2,
	}
	config.Cache = CacheConfig{
		Path:          "./cache",
		MaxCacheSize:  300,
		KeepFavorites: false,
	}
	config.Db = DbConfig{
		Path: "./data.db",
	}
}

var configMutex = sync.Mutex{}

func LoadConfig() {
	FillDefaultSetting()
	currentVersion := config.Version

	_, err := os.Stat("config.yaml")
	if errors.Is(err, os.ErrPermission) {
		log.Fatalln("config.yaml permission denied")
	}

	if err == nil {
		configFile, err := os.Open("config.yaml")
		if err != nil {
			log.Fatalf("open config.yaml error: %s", err)
		}
		defer configFile.Close()

		decoder := yaml.NewDecoder(configFile)
		err = decoder.Decode(&config)
		if err != nil {
			log.Fatalf("Failed to parse config.yaml: %s", err)
		}
	}

	if config.Version != currentVersion {
		// TODO show features
	}

	checkInterceptionConflict()
	checkPreloadConflict()

	SaveConfig()
}

func SaveConfig() {
	configMutex.Lock()
	defer configMutex.Unlock()

	configFile, err := os.Create("config.yaml")
	if err != nil {
		log.Fatalf("Open or create config.yaml error: %s", err)
	}
	defer configFile.Close()

	encoder := yaml.NewEncoder(configFile)
	err = encoder.Encode(&config)
	if err != nil {
		log.Fatalf("Failed to save config.yaml: %s", err)
	}
}

func GetHijackConfig() *HijackConfig {
	return &config.Hijack
}
func GetKeyConfig() *KeyConfig {
	return &config.Keys
}
func GetProxyConfig() *ProxyConfig {
	return &config.Proxy
}
func GetPreloadConfig() *PreloadConfig {
	return &config.Preload
}
func GetDownloadConfig() *DownloadConfig {
	return &config.Download
}
func GetCacheConfig() *CacheConfig {
	return &config.Cache
}
func GetDbConfig() *DbConfig {
	return &config.Db
}
