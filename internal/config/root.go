package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/downloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/live"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/mixed_server"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/preloader"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/secrets"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
)

type Config struct {
	Version string `yaml:"version"`

	Server mixed_server.Config `yaml:"main-server"`
	Proxy  requesting.Config   `yaml:"proxy"`

	Secrets secrets.Config `yaml:"-"`

	ThirdParties third_parties.Config     `yaml:"third-parties"`
	Executable   local_executables.Config `yaml:"executable"`

	Preloader  preloader.Config  `yaml:"preloader"`
	Downloader downloader.Config `yaml:"downloader"`

	Cache cache_manager.Config `yaml:"cache"`

	Live live.Config `yaml:"live-server"`
}

func Default() Config {
	secretConfig, err := secrets.DefaultConfig()
	if err != nil {
		panic(err)
	}

	return Config{
		Version: "3.0",

		Server: mixed_server.DefaultConfig(),
		Proxy:  requesting.DefaultConfig(),

		Secrets: secretConfig,

		ThirdParties: third_parties.DefaultConfig(),
		Executable:   local_executables.DefaultConfig(),

		Preloader:  preloader.DefaultConfig(),
		Downloader: downloader.DefaultConfig(),

		Cache: cache_manager.DefaultConfig(),

		Live: live.DefaultConfig(),
	}
}
