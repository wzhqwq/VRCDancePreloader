package host

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/config/migrated_config"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/live"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/mixed_server"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
)

var currentManager *Manager

func StartHost() bool {
	cfg, migrated := migrated_config.LoadLatestConfig()

	currentManager = NewManager(cfg)
	err := Register(currentManager)
	if err != nil {
		logger.FatalLn("Failed to register services:", err)
	}
	currentManager.Start()

	return migrated
}

func Shutdown() {
	currentManager.GracefulShutdown()
}

func Config() *config.Manager {
	return currentManager.Config()
}

func MixedServer() *mixed_server.Service {
	svc, ok := currentManager.GetService("main_server").(*mixed_server.Service)
	if !ok {
		return nil
	}
	return svc
}

func LiveServer() *live.Service {
	svc, ok := currentManager.GetService("live_server").(*live.Service)
	if !ok {
		return nil
	}
	return svc
}

func Downloader() *downloader.Service {
	svc, ok := currentManager.GetService("downloader").(*downloader.Service)
	if !ok {
		return nil
	}
	return svc
}

func Preloader() *preloader.Service {
	svc, ok := currentManager.GetService("preloader").(*preloader.Service)
	if !ok {
		return nil
	}
	return svc
}

func CacheManager() *cache_manager.Service {
	svc, ok := currentManager.GetService("cache_manager").(*cache_manager.Service)
	if !ok {
		return nil
	}
	return svc
}
