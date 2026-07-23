package migrated_config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/secrets"
	"gopkg.in/yaml.v3"
)

var configV2 struct {
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

type HijackConfig struct {
	ProxyPort        int      `yaml:"proxy-port"`
	InterceptedSites []string `yaml:"intercepted-sites"`
	EnableHttps      bool     `yaml:"enable-https"`
	EnablePWI        bool     `yaml:"enable-pwi"`
	LimitBandwidth   bool     `yaml:"limit-bandwidth"`
	MaxBandwidth     int      `yaml:"max-bandwidth"`
}

type ProxyConfig struct {
	Pypy  string `yaml:"pypydance-api"`
	Wanna string `yaml:"wannadance-api"`
	DuDu  string `yaml:"dudu-fitdance-api"`

	BiliBiliAPI   string `yaml:"bilibili-api"`
	BiliBiliVideo string `yaml:"bilibili-video"`

	YoutubeVideo string `yaml:"youtube-video"`
	YoutubeApi   string `yaml:"youtube-api"`
	YoutubeImage string `yaml:"youtube-image"`

	GitHubApi    string `yaml:"github-api"`
	GitHubAssets string `yaml:"github-assets"`
}

type KeyConfig struct {
	Youtube string `yaml:"youtube-api"`
}

type YoutubeConfig struct {
	EnableApi       bool `yaml:"enable-youtube-api"`
	EnableThumbnail bool `yaml:"enable-youtube-thumbnail"`
	EnableYtDlp     bool `yaml:"enable-ytdlp"`
}

type ExecutableConfig struct {
	CheckUpdateOnStart bool `yaml:"check_update_on_start"`

	YtDlpPath string `yaml:"yt-dlp-path"`
	DenoPath  string `yaml:"deno-path"`
}

type PreloadConfig struct {
	EnabledRooms     []string `yaml:"enabled-rooms"`
	EnabledPlatforms []string `yaml:"enabled-platforms"`
	MaxPreload       int      `yaml:"max-preload-count"`
}

type DownloadConfig struct {
	MaxDownload int `yaml:"max-parallel-download-count"`
}

type CacheConfig struct {
	Path          string `yaml:"path"`
	MaxCacheSize  int    `yaml:"max-cache-size"`
	KeepFavorites bool   `yaml:"keep-favorites"`
	//RWBufferSize  int    `yaml:"rw-buffer-size"`
	// 0: legacy, 1: continuous, 2: fragmented
	FileFormat int `yaml:"file-format"`

	ForceExpirationCheck bool `yaml:"force-expiration-check"`
}

type DbConfig struct {
	Path string `yaml:"path"`
}

type LiveConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Port     int    `yaml:"port"`
	Settings string `yaml:"settings"`
}

func fillDefaultSetting() {
	configV2.Version = "2.2"
	configV2.Hijack = HijackConfig{
		ProxyPort:        7653,
		InterceptedSites: constants.CopyAllSites(),
	}
	configV2.Proxy = ProxyConfig{}
	configV2.Keys = KeyConfig{}
	configV2.Youtube = YoutubeConfig{}
	configV2.Executable = ExecutableConfig{
		YtDlpPath: "<vrcdp>",
		DenoPath:  "<vrcdp>",
	}
	configV2.Preload = PreloadConfig{
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
	configV2.Download = DownloadConfig{
		MaxDownload: 1,
	}
	configV2.Cache = CacheConfig{
		Path:         "./cache",
		MaxCacheSize: 300,
		FileFormat:   1,
	}
	configV2.Db = DbConfig{
		Path: "./data.db",
	}
	configV2.Live = LiveConfig{
		Port:     7652,
		Settings: "{}",
	}
}

func loadConfig() {
	fillDefaultSetting()

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
		err = decoder.Decode(&configV2)
		if err != nil {
			logger.FatalLnf("Failed to parse config.yaml: %s", err)
		}
	}
}

func moveFolder(oldPath, newPath string) error {
	err := os.MkdirAll(oldPath, os.ModePerm)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(oldPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		oldChild := path.Join(oldPath, f.Name())
		newChild := path.Join(newPath, f.Name())
		if f.IsDir() {
			err = moveFolder(oldChild, newChild)
			if err != nil {
				return err
			}
		} else {
			err = os.Rename(oldChild, newChild)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func migrateDB() error {
	permanentDbPath := filepath.Join(custom_fyne.AppDataRoot, "db", "data.db")
	lowDbFolderPath := filepath.Join(custom_fyne.LowAppDataRoot, "db")

	_, err := os.Stat(permanentDbPath)

	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(permanentDbPath), 0755)
		if err != nil {
			return err
		}

		if _, err = os.Stat(lowDbFolderPath); !os.IsNotExist(err) {
			if err != nil {
				return err
			}
			return moveFolder(lowDbFolderPath, filepath.Dir(permanentDbPath))
		}

		oldPath := configV2.Db.Path

		if oldPath != "" {
			if _, err = os.Stat(oldPath); !os.IsNotExist(err) {
				if err != nil {
					return err
				}

				file, err := os.Open(oldPath)
				if err != nil {
					return err
				}
				defer file.Close()

				newFile, err := os.Create(permanentDbPath)
				if err != nil {
					return err
				}
				defer newFile.Close()

				_, err = io.Copy(newFile, file)
				if err != nil {
					return err
				}

				err = newFile.Sync()
				if err != nil {
					return err
				}

				logger.InfoLn("Migrated database to", permanentDbPath)
				MigrationNotes = append(MigrationNotes, i18n.T("message_db_migration", goeasyi18n.Options{
					Data: map[string]interface{}{
						"Dir": permanentDbPath,
					},
				}))
			}
		}
	}

	return nil
}

func tryApply(cfg *config.Config, modifier func(c *config.Config) error) error {
	cfgNew := *cfg
	if err := modifier(&cfgNew); err != nil {
		return err
	}
	*cfg = cfgNew
	return nil
}

func MigrateFromV2(cfg *config.Config) error {
	loadConfig()

	var AllErrors []error

	if err := migrateDB(); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to migrate DB: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		c.Server.Port = configV2.Hijack.ProxyPort
		c.Server.EnableHttpsProxy = configV2.Hijack.EnableHttps
		c.Server.InterceptedSites = configV2.Hijack.InterceptedSites
		return c.Server.Validate()
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new server config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		c.Proxy.Pypy = configV2.Proxy.Pypy
		c.Proxy.Wanna = configV2.Proxy.Wanna
		c.Proxy.DuDu = configV2.Proxy.DuDu
		c.Proxy.BiliBili = configV2.Proxy.BiliBiliAPI
		c.Proxy.YoutubeVideo = configV2.Proxy.YoutubeVideo
		c.Proxy.YoutubeApi = configV2.Proxy.YoutubeApi
		c.Proxy.YoutubeImage = configV2.Proxy.YoutubeImage
		c.Proxy.GitHubApi = configV2.Proxy.GitHubApi
		c.Proxy.GitHubAssets = configV2.Proxy.GitHubAssets
		return c.Proxy.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new proxy config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		c.Secrets.YoutubeApiKey = configV2.Keys.Youtube
		err := c.Secrets.Validate("")
		if err != nil {
			return err
		}
		return secrets.Migrate(c.Secrets)
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new secrets config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		if configV2.Youtube.EnableYtDlp {
			c.ThirdParties.YoutubeMode = "ytdlp"
		} else if configV2.Youtube.EnableApi {
			c.ThirdParties.YoutubeMode = "api"
		}

		// In v2, thumbnails of BiliBili video and PyPyDance video are loaded by default
		c.ThirdParties.BiliBiliResources = []string{"video", "info", "thumbnail"}
		c.ThirdParties.PyPyDanceResources = []string{"video", "catalog", "thumbnail"}
		if configV2.Youtube.EnableThumbnail {
			c.ThirdParties.YoutubeResources = []string{"video", "info", "thumbnail"}
		}
		return c.ThirdParties.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new third party config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		c.Executable.CheckUpdateOnStart = configV2.Executable.CheckUpdateOnStart
		c.Executable.YtDlpPath = configV2.Executable.YtDlpPath
		c.Executable.DenoPath = configV2.Executable.DenoPath
		return c.Executable.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new executable config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		// "enabled-rooms" and "enabled-platforms" are not actually used in v2
		c.Preloader.MaxPreload = configV2.Preload.MaxPreload
		return c.Preloader.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new preload config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		c.Downloader.MaxDownload = configV2.Download.MaxDownload
		return c.Downloader.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new download config: %s", err))
	}

	if err := tryApply(cfg, func(c *config.Config) error {
		if configV2.Cache.FileFormat < 1 {
			logger.WarnLn("We no longer support writing legacy cache files. `cache.video-file-format` will be replaced with default value")
			configV2.Cache.FileFormat = 1
		}
		c.Cache.MaxVideoCache = configV2.Cache.MaxCacheSize
		c.Cache.Path = configV2.Cache.Path
		c.Cache.VideoFileFormat = configV2.Cache.FileFormat
		c.Cache.KeepFavorites = configV2.Cache.KeepFavorites
		c.Cache.ForceExpiration = configV2.Cache.ForceExpirationCheck
		return c.Cache.Validate("")
	}); err != nil {
		AllErrors = append(AllErrors, fmt.Errorf("failed to validate new cache config: %s", err))
	}

	if len(AllErrors) > 0 {
		return fmt.Errorf("migration failures: %w", errors.Join(AllErrors...))
	}
	return nil
}
