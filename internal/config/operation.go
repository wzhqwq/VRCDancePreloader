package config

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/hijack"
	"github.com/wzhqwq/VRCDancePreloader/internal/live"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

func (hc *HijackConfig) Init() {
	runner := input.NewServerRunner(hc.ProxyPort)
	runner.OnSave = hc.UpdatePort
	runner.StartServer = func() error {
		if err := hijack.Start(hc.InterceptedSites, hc.EnableHttps, hc.ProxyPort); err != nil {
			if global_state.IsInGui() {
				return err
			}

			logger.FatalLn("Failed to start hijack server:", err)
		}
		return nil
	}
	runner.StopServer = config.Hijack.Stop
	runner.Run()

	hc.HijackRunner = runner
	if hc.EnablePWI {
		service.StartPWIServer()
	}
}

func (hc *HijackConfig) Stop() {
	hijack.Stop()
	if hc.EnablePWI {
		service.StopPWIServer()
	}
}

func (hc *HijackConfig) UpdatePort(port int) {
	hc.ProxyPort = port
	SaveConfig()
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
	SaveConfig()
}

func (hc *HijackConfig) UpdateLimitBandwidth(b bool) {
	hc.LimitBandwidth = b
	hijack.SetLimitBandwidth(b)
	SaveConfig()
}

func (pc *ProxyConfig) Init() {
	//TODO cancel comment after implemented youtube preloading
	pc.ProxyControllers = map[string]*ProxyTester{
		"pypydance-api":     NewProxyTester("pypydance-api", pc.Pypy),
		"wannadance-api":    NewProxyTester("wannadance-api", pc.Wanna),
		"dudu-fitdance-api": NewProxyTester("dudu-fitdance-api", pc.DuDu),
		"bilibili-api":      NewProxyTester("bilibili-api", pc.BiliBili),
		"youtube-video":     NewProxyTester("youtube-video", pc.YoutubeVideo),
		"youtube-api":       NewProxyTester("youtube-api", pc.YoutubeApi),
		"youtube-image":     NewProxyTester("youtube-image", pc.YoutubeImage),
	}

	requesting.InitClient(requesting.PyPyDance, pc.Pypy)
	requesting.InitClient(requesting.WannaDance, pc.Wanna)
	requesting.InitClient(requesting.DuDuFitDance, pc.DuDu)
	requesting.InitClient(requesting.BiliBiliApi, pc.BiliBili)
	requesting.InitClient(requesting.YouTubeVideo, pc.YoutubeVideo)
	requesting.InitClient(requesting.YouTubeImage, pc.YoutubeImage)
	requesting.InitClient(requesting.YouTubeApi, pc.YoutubeApi)

	if !skipTest {
		pc.ProxyControllers["pypydance-api"].Test()
		pc.ProxyControllers["wannadance-api"].Test()
		pc.ProxyControllers["dudu-fitdance-api"].Test()
		pc.ProxyControllers["bilibili-api"].Test()
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

func (pc *ProxyConfig) Update(item, value string) error {
	_, err := url.Parse(value)
	if err != nil {
		return err
	}

	switch item {
	case "pypydance-api":
		pc.Pypy = value
		requesting.UpdateClient(requesting.PyPyDance, value)
	case "wannadance-api":
		pc.Wanna = value
		requesting.UpdateClient(requesting.WannaDance, value)
	case "dudu-fitdance-api":
		pc.DuDu = value
		requesting.UpdateClient(requesting.DuDuFitDance, value)
	case "bilibili-api":
		pc.BiliBili = value
		requesting.UpdateClient(requesting.BiliBiliApi, value)
	case "youtube-video":
		pc.YoutubeVideo = value
		requesting.UpdateClient(requesting.YouTubeVideo, value)
	case "youtube-api":
		pc.YoutubeApi = value
		requesting.UpdateClient(requesting.YouTubeApi, value)
	case "youtube-image":
		pc.YoutubeImage = value
		requesting.UpdateClient(requesting.YouTubeImage, value)
	default:
		logger.FatalLnf("Unknown proxy item: %s", item)
	}
	SaveConfig()
	return nil
}

func (pc *ProxyConfig) Test(item string) (bool, string) {
	switch item {
	case "pypydance-api":
		return requesting.TestClient(requesting.PyPyDance)
	case "wannadance-api":
		return requesting.TestClient(requesting.WannaDance)
	case "dudu-fitdance-api":
		return requesting.TestClient(requesting.DuDuFitDance)
	case "bilibili-api":
		return requesting.TestClient(requesting.BiliBiliApi)
	case "youtube-video":
		return requesting.TestClient(requesting.YouTubeVideo)
	case "youtube-api":
		return requesting.TestClient(requesting.YouTubeApi)
	case "youtube-image":
		return requesting.TestClient(requesting.YouTubeImage)
	default:
		logger.FatalLnf("Unknown proxy item: %s", item)
		return false, ""
	}
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
	if cc.FileFormat <= 0 {
		logger.WarnLn("We no longer support writing legacy cache files. `cache.file-format` will be replaced with default value")
		cc.FileFormat = 1
		SaveConfig()
	}

	cache.SetupCache(cc.Path)
	cache.SetMaxSize(int64(cc.MaxCacheSize) * 1024 * 1024)
	cache.SetKeepFavorites(cc.KeepFavorites)
	entry.SetFileFormat(cc.FileFormat)
	entry.SetForceExpirationCheck(cc.ForceExpirationCheck)
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

func (cc *CacheConfig) UpdateForceExpirationCheck(b bool) {
	cc.ForceExpirationCheck = b
	entry.SetForceExpirationCheck(b)
	SaveConfig()
}

func (cc *CacheConfig) UpdateFileFormat(fileFormat int) {
	cc.FileFormat = fileFormat
	entry.SetFileFormat(fileFormat)
	SaveConfig()
}

func (dc *DbConfig) Init() error {
	// migration

	// AppData copied from ytdlp-test branch
	const AppName = "VRCDP"
	configDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	AppDataRoot := ""
	if strings.HasSuffix(configDir, "Roaming") {
		// it's Windows, and we should store large data to LocalLow
		AppDataRoot = filepath.Join(configDir, "..", "LocalLow", AppName)
	} else {
		AppDataRoot = filepath.Join(configDir, AppName, "data")
	}

	permanentDbPath := filepath.Join(AppDataRoot, "db", "data.db")
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

	err = persistence.InitDB(permanentDbPath)
	if err != nil {
		return err
	}
	return nil
}

func (lc *LiveConfig) Init() {
	live.OnSettingsChanged = func(settings string) {
		lc.UpdateSettings(settings)
	}
	live.GetSettings = func() string {
		return lc.Settings
	}

	runner := input.NewServerRunner(lc.Port)
	runner.OnSave = lc.UpdatePort
	runner.StartServer = func() error {
		if err := live.StartLiveServer(lc.Port); err != nil {
			if global_state.IsInGui() {
				return err
			}

			logger.FatalLn("Failed to start live server:", err)
		}
		return nil
	}
	runner.StopServer = config.Hijack.Stop
	lc.LiveRunner = runner

	if lc.Enabled {
		runner.Run()
	}
}

func (lc *LiveConfig) UpdateEnable(b bool) {
	lc.Enabled = b
	if lc.Enabled {
		lc.LiveRunner.Run()
	} else {
		live.StopLiveServer()
	}
	SaveConfig()
}

func (lc *LiveConfig) UpdatePort(port int) {
	lc.Port = port
	SaveConfig()
}

func (lc *LiveConfig) UpdateSettings(settings string) {
	lc.Settings = settings
	SaveConfig()
}

func (lc *LiveConfig) Stop() {
	live.StopLiveServer()
}
