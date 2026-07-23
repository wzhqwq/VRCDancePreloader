package host

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/live"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/mixed_server"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/secrets"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/catalog"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/watcher"
)

func Register(m *Manager) error {
	// Basic long-term services

	m.RegisterService(
		"proxy",
		requesting.New(m.cfg.Proxy),
		OneFieldCfgApplier[requesting.Config]{
			func(cfg config.Config) requesting.Config {
				return cfg.Proxy
			},
			requesting.UpdateConfig,
		},
	)

	m.RegisterService(
		"secrets",
		secrets.New(m.cfg.Secrets),
		OneFieldCfgApplier[secrets.Config]{
			func(cfg config.Config) secrets.Config {
				return cfg.Secrets
			},
			secrets.UpdateConfig,
		},
	)

	m.RegisterServiceWithoutConfig("db", persistence.New())
	m.RegisterServiceWithoutConfig("playlist", playlist.New(), "db")
	m.RegisterServiceWithoutConfig("watcher", watcher.New(), "playlist", "db")

	// CDN-related services

	cacheManagerSvc := cache_manager.New(m.cfg.Cache)
	m.RegisterService(
		"cache_manager",
		cacheManagerSvc,
		OneFieldCfgApplier[cache_manager.Config]{
			func(cfg config.Config) cache_manager.Config {
				return cfg.Cache
			},
			cacheManagerSvc.UpdateConfig,
		},
		// "db" for saving metadata
		"db",
	)

	// Third-party tools

	m.RegisterService(
		"executable",
		local_executables.New(m.cfg.Executable),
		OneFieldCfgApplier[local_executables.Config]{
			func(cfg config.Config) local_executables.Config {
				return cfg.Executable
			},
			local_executables.UpdateConfig,
		},
		// "proxy" for downloading executables
		"proxy",
	)

	m.RegisterServiceWithoutConfig(
		"catalog",
		catalog.New(cacheManagerSvc),
		// "proxy" for downloading catalog, "cache_manager" for caching catalog
		"proxy", "cache_manager",
	)

	m.RegisterService(
		"third_parties",
		third_parties.New(m.cfg.ThirdParties),
		OneFieldCfgApplier[third_parties.Config]{
			func(cfg config.Config) third_parties.Config {
				return cfg.ThirdParties
			},
			third_parties.UpdateConfig,
		},
		// "proxy" for requesting api, "secrets" for api key, "executable" for yt-dlp and deno, "catalog" for dance room catalog
		"proxy", "secrets", "executable", "catalog",
	)

	// Playlist-related services

	downloaderSvc := downloader.New(m.cfg.Downloader)
	m.RegisterService(
		"downloader",
		downloaderSvc,
		OneFieldCfgApplier[downloader.Config]{
			func(cfg config.Config) downloader.Config {
				return cfg.Downloader
			},
			downloaderSvc.UpdateConfig,
		},
	)

	preloaderSvc := preloader.New(m.cfg.Preloader, cacheManagerSvc, downloaderSvc)
	m.RegisterService(
		"preloader",
		preloaderSvc,
		OneFieldCfgApplier[preloader.Config]{
			func(cfg config.Config) preloader.Config {
				return cfg.Preloader
			},
			preloaderSvc.UpdateConfig,
		},
		// "third_parties" for video resource info, "playlist" for selecting songs to be downloaded
		// "cache_manager" and "downloader" for downloading songs
		"third_parties", "playlist", "cache_manager", "downloader",
	)

	// File, hijacking, h5 UI services

	serverSvc := mixed_server.New(m.cfg.Server, preloaderSvc)
	m.RegisterService(
		"main_server",
		serverSvc,
		NoFieldCfgApplier[mixed_server.Config]{
			func(cfg config.Config) mixed_server.Config {
				return cfg.Server
			},
			serverSvc.UpdateConfig,
		},
		// "preloader" for finding song and trigger download
		"preloader",
	)

	liveSvc := live.New(m.cfg.Live)
	m.RegisterService(
		"live_server",
		liveSvc,
		NoFieldCfgApplier[live.Config]{
			func(cfg config.Config) live.Config {
				return cfg.Live
			},
			liveSvc.UpdateConfig,
		},
		// "db" for frontend settings
		"db",
	)

	return nil
}
