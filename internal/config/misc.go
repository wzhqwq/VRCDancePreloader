package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/local_executables"
)

type KeyConfig struct {
	Youtube string `yaml:"youtube-api"`
}

func GetKeyConfig() *KeyConfig {
	return &config.Keys
}

func (kc *KeyConfig) Init() {
	if config.Youtube.EnableApi {
		if kc.Youtube != "" {
			third_party_api.YoutubeApiKey = kc.Youtube
		} else {
			logger.WarnLn("YouTube API feature is disabled because YouTube API key is missing")
			config.Youtube.UpdateEnableApi(false)
		}
	}
}

type ExecutableConfig struct {
	CheckUpdateOnStart bool `yaml:"check_update_on_start"`

	YtDlpPath string `yaml:"yt-dlp-path"`
	DenoPath  string `yaml:"deno-path"`
}

func GetExecutableConfig() *ExecutableConfig {
	return &config.Executable
}

func (ec *ExecutableConfig) Init() {
	local_executables.InitYtDlp(ec.CheckUpdateOnStart, ec.YtDlpPath)
	local_executables.InitDeno(ec.CheckUpdateOnStart, ec.DenoPath)
}

func (ec *ExecutableConfig) UpdateCheckUpdateOnStart(enable bool) {
	ec.CheckUpdateOnStart = enable
	SaveConfig()
}

func (ec *ExecutableConfig) UpdateYtDlpPath(p string) {
	ec.YtDlpPath = p
	SaveConfig()
	local_executables.InitYtDlp(ec.CheckUpdateOnStart, ec.YtDlpPath)
}

func (ec *ExecutableConfig) UpdateDenoPath(p string) {
	ec.DenoPath = p
	SaveConfig()
	local_executables.InitDeno(ec.CheckUpdateOnStart, ec.DenoPath)
}

type PreloadConfig struct {
	EnabledRooms     []string `yaml:"enabled-rooms"`
	EnabledPlatforms []string `yaml:"enabled-platforms"`
	MaxPreload       int      `yaml:"max-preload-count"`
}

func GetPreloadConfig() *PreloadConfig {
	return &config.Preload
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

type DownloadConfig struct {
	MaxDownload int `yaml:"max-parallel-download-count"`
}

func GetDownloadConfig() *DownloadConfig {
	return &config.Download
}

func (dc *DownloadConfig) Init() {
	download.InitDownloadManager(dc.MaxDownload)
}

func (dc *DownloadConfig) UpdateMaxDownload(max int) {
	dc.MaxDownload = max
	download.SetMaxParallel(max)
	SaveConfig()
}

type DbConfig struct {
	Path string `yaml:"path"`
}

func GetDbConfig() *DbConfig {
	return &config.Db
}

func (dc *DbConfig) Init() error {
	// migration
	permanentDbPath := filepath.Join(custom_fyne.AppDataRoot, "db", "data.db")
	if _, err := os.Stat(permanentDbPath); os.IsNotExist(err) {
		err := os.MkdirAll(filepath.Dir(permanentDbPath), 0755)
		if err != nil {
			return err
		}
		if dc.Path != "" {
			if _, err = os.Stat(dc.Path); err == nil {
				file, err := os.Open(dc.Path)
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
				global_state.SetDbMigrationPath(permanentDbPath)
			}
		}
	}

	err := persistence.InitDB(permanentDbPath)
	if err != nil {
		return err
	}
	return nil
}
