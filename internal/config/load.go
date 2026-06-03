package config

import (
	"errors"
	"os"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"gopkg.in/yaml.v3"
)

var logger = utils.NewLogger("Config File")

var em = utils.NewEventManager[string]()

var config struct {
	Version string `yaml:"version"`

	Hijack HijackConfig `yaml:"hijack"`
	Proxy  ProxyConfig  `yaml:"proxy"`

	Keys       KeyConfig        `yaml:"keys"`
	Youtube    YoutubeConfig    `yaml:"youtube"`
	Executable ExecutableConfig `yaml:"executable"`

	Preload  PreloadConfig  `yaml:"preload"`
	Download DownloadConfig `yaml:"download"`

	Cache CacheConfig `yaml:"cache"`
	Db    DbConfig    `yaml:"db"`

	Live LiveConfig `yaml:"live"`
}

func FillDefaultSetting() {
	config.Version = "2.2"
	config.Hijack = defaultHijackConfig
	config.Proxy = defaultProxyConfig
	config.Keys = KeyConfig{}
	config.Youtube = defaultYoutubeConfig
	config.Executable = ExecutableConfig{
		YtDlpPath: "<vrcdp>",
		DenoPath:  "<vrcdp>",
	}
	config.Preload = PreloadConfig{
		EnabledRooms: []string{
			"PyPyDance",
			"WannaDance",
			"DuDuFitDance",
		},
		EnabledPlatforms: []string{
			"PyPyDance",
			"WannaDance",
			"DuDuFitDance",
			"BiliBili",
			//"YouTube",
		},
		MaxPreload: 2,
	}
	config.Download = DownloadConfig{
		MaxDownload: 1,
	}
	config.Cache = defaultCacheConfig
	config.Db = DbConfig{
		Path: "./data.db",
	}
	config.Live = defaultLiveConfig
}

var configMutex = sync.Mutex{}

func LoadConfig() {
	FillDefaultSetting()
	currentVersion := config.Version

	_, err := os.Stat("config.yaml")
	if errors.Is(err, os.ErrPermission) {
		logger.FatalLn("config.yaml permission denied")
	}

	if err == nil {
		configFile, err := os.Open("config.yaml")
		if err != nil {
			logger.FatalLnf("Open config.yaml error: %s", err)
		}
		defer configFile.Close()

		decoder := yaml.NewDecoder(configFile)
		err = decoder.Decode(&config)
		if err != nil {
			logger.FatalLnf("Failed to parse config.yaml: %s", err)
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
		logger.FatalLnf("Open or create config.yaml error: %s", err)
	}
	defer configFile.Close()

	encoder := yaml.NewEncoder(configFile)
	err = encoder.Encode(&config)
	if err != nil {
		logger.FatalLnf("Failed to save config.yaml: %s", err)
	}
}

func saveAndNotify(category string) {
	SaveConfig()
	em.NotifySubscribers(category)
}

func Subscribe() *utils.EventSubscriber[string] {
	return em.SubscribeEvent()
}
