package host

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/config/migrated_config"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/live"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/mixed_server"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/stability"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var currentManager *Manager

func StartHost() bool {
	cfg, migrated := migrated_config.LoadLatestConfig()

	currentManager = NewManager(cfg)
	err := Register(currentManager)
	if err != nil {
		logger.FatalLn("Failed to register services:", err)
	}
	currentManager.Init()
	currentManager.Start()

	return migrated
}

func Shutdown() {
	cancel := stability.PanicIfTimeout("Host_ShuttingDown")
	defer cancel()
	currentManager.GracefulShutdown()
}

func Config() *config.Manager {
	return currentManager.Config()
}

func Status() interactive.RunnerStatus {
	return currentManager.Status()
}

func MixedServer() *mixed_server.Service {
	svc, ok := currentManager.getService("main_server").(*mixed_server.Service)
	if !ok {
		return nil
	}
	return svc
}

func LiveServer() *live.Service {
	svc, ok := currentManager.getService("live_server").(*live.Service)
	if !ok {
		return nil
	}
	return svc
}

func Downloader() *downloader.Service {
	svc, ok := currentManager.getService("downloader").(*downloader.Service)
	if !ok {
		return nil
	}
	return svc
}

func Preloader() *preloader.Service {
	svc, ok := currentManager.getService("preloader").(*preloader.Service)
	if !ok {
		return nil
	}
	return svc
}

func CacheManager() *cache_manager.Service {
	svc, ok := currentManager.getService("cache_manager").(*cache_manager.Service)
	if !ok {
		return nil
	}
	return svc
}
