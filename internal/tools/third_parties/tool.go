package third_parties

import (
	"strings"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cfg Config

var providers map[string]ResourceProvider

var logger = utils.NewLogger("Third Parties")

func GetProviderById(id string) ResourceProvider {
	split := strings.Split(id, "_")
	if len(split) != 2 {
		return nil
	}
	return providers[split[0]]
}

func setAll(cfg Config) {
	providers["yt"].SetMode(cfg.YoutubeMode)
	providers["bili"].SetMode(cfg.BiliBiliMode)
	providers["yt"].SetAllowResources(cfg.YoutubeResources)
	providers["bili"].SetAllowResources(cfg.BiliBiliResources)
	providers["pypy"].SetAllowResources(cfg.PyPyDanceResources)
	providers["wanna"].SetAllowResources(cfg.WannaDanceResources)
	providers["dudu"].SetAllowResources(cfg.DuDuFitDanceResources)
}

func initialize() error {
	providers = map[string]ResourceProvider{
		"yt":    newYouTubeProvider(),
		"bili":  newBiliBiliProvider(),
		"pypy":  newPyPyDanceProvider(),
		"wanna": newWannaDanceProvider(),
		"dudu":  newDuDuFitDanceProvider(),
	}
	setAll(cfg)
	for _, provider := range providers {
		provider.Start()
	}

	return nil
}

func destroy() error {
	for _, provider := range providers {
		provider.Close()
	}
	return nil
}

type Tool struct {
	service.ConfigurableTool
}

func New(c Config) *Tool {
	cfg = c

	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool("third_parties", initialize, destroy, logger),
	}
}

func UpdateConfig(c Config, field string) error {
	if field == "" {
		setAll(c)
	} else {
		switch field {
		case keyYtMode:
			providers["yt"].SetMode(c.YoutubeMode)
		case keyBiliMode:
			providers["bili"].SetMode(c.BiliBiliMode)

		case keyYtAllowedResources:
			providers["yt"].SetAllowResources(c.YoutubeResources)
		case keyBiliAllowedResources:
			providers["bili"].SetAllowResources(c.BiliBiliResources)
		case keyPypyAllowedResources:
			providers["pypy"].SetAllowResources(c.PyPyDanceResources)
		case keyWannaAllowedResources:
			providers["wanna"].SetAllowResources(c.WannaDanceResources)
		case keyDuduAllowedResources:
			providers["dudu"].SetAllowResources(c.DuDuFitDanceResources)
		}
	}

	cfg = c
	return nil
}
