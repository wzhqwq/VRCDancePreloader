package config

import (
	"embed"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

//go:embed config_template.yaml
var templateConfigFileFS embed.FS

type KeyConfig struct {
	Youtube string `yaml:"youtube-api"`
}
type ProxyConfig struct {
	Pypy         string `yaml:"jd.pypy.moe"`
	YoutubeVideo string `yaml:"youtube-video"`
	YoutubeApi   string `yaml:"youtube-api"`
	YoutubeImage string `yaml:"youtube-image"`
}
type YoutubeConfig struct {
	EnableApi       bool `yaml:"enable-youtube-api"`
	EnableThumbnail bool `yaml:"enable-youtube-thumbnail"`
	EnableVideo     bool `yaml:"enable-youtube-video"`
}
type PreloadConfig struct {
	MaxPreload int `yaml:"max-preload-count"`
}
type DownloadConfig struct {
	MaxDownload int `yaml:"max-parallel-download-count"`
}
type CacheConfig struct {
	Path          string   `yaml:"path"`
	MaxCacheSize  int      `yaml:"max-cache-size"`
	KeepFavorites bool     `yaml:"keep-favorites"`
	Whitelist     []string `yaml:"whitelist"`
}

var config struct {
	Proxy    ProxyConfig    `yaml:"proxy"`
	Keys     KeyConfig      `yaml:"keys"`
	Youtube  YoutubeConfig  `yaml:"youtube"`
	Preload  PreloadConfig  `yaml:"preload"`
	Download DownloadConfig `yaml:"download"`
	Cache    CacheConfig    `yaml:"cache"`
}

func CreateIfNotExists() {
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		templateFile, err := templateConfigFileFS.ReadFile("config_template.yaml")
		if err != nil {
			log.Fatal(err)
		}

		configFile, err := os.Create("config.yaml")
		if err != nil {
			log.Fatal(err)
		}
		defer configFile.Close()

		if _, err := configFile.Write(templateFile); err != nil {
			log.Fatal(err)
		}

		log.Println("Created config.yaml, you can customize the preloader by editing it")
	}
}

func LoadConfig() {
	CreateIfNotExists()

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
