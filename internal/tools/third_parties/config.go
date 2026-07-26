package third_parties

import (
	"fmt"
)

type Config struct {
	// disabled, api, ytdlp
	YoutubeMode  string `yaml:"youtube-mode"`
	BiliBiliMode string `yaml:"bilibili-mode"`

	// video, info, thumbnail
	YoutubeResources  []string `yaml:"youtube-allow-resources"`
	BiliBiliResources []string `yaml:"bilibili-allow-resources"`

	// video, catalog, thumbnail
	PyPyDanceResources    []string `yaml:"pypydance-allow-resources"`
	WannaDanceResources   []string `yaml:"wannadance-allow-resources"`
	DuDuFitDanceResources []string `yaml:"dudu-fitdance-allow-resources"`
}

const keyYtMode = "youtube-mode"
const keyBiliMode = "bilibili-mode"
const keyYtAllowedResources = "youtube-allow-resources"
const keyBiliAllowedResources = "bilibili-allow-resources"
const keyPypyAllowedResources = "pypydance-allow-resources"
const keyWannaAllowedResources = "wannadance-allow-resources"
const keyDuduAllowedResources = "dudu-fitdance-allow-resources"

func DefaultConfig() Config {
	return Config{
		YoutubeMode:  "disabled",
		BiliBiliMode: "disabled",

		YoutubeResources:  []string{"video", "info"},
		BiliBiliResources: []string{"video", "info"},

		PyPyDanceResources:    []string{"video", "catalog"},
		WannaDanceResources:   []string{"video", "catalog"},
		DuDuFitDanceResources: []string{"video", "catalog"},
	}
}

func (c Config) Validate(field string) error {
	if field == "" {
		if !ValidMode(c.YoutubeMode) {
			return fmt.Errorf("invalid YouTube mode: %s", c.YoutubeMode)
		}
		if !ValidMode(c.BiliBiliMode) {
			return fmt.Errorf("invalid BiliBili mode: %s", c.BiliBiliMode)
		}
		if !ValidPlatformResources(c.YoutubeResources) {
			return fmt.Errorf("invalid YouTube resources: %s", c.YoutubeResources)
		}
		if !ValidPlatformResources(c.BiliBiliResources) {
			return fmt.Errorf("invalid BiliBili resources: %s", c.BiliBiliResources)
		}
		if !ValidRoomResources(c.PyPyDanceResources) {
			return fmt.Errorf("invalid PyPyDance resources: %s", c.PyPyDanceResources)
		}
		if !ValidRoomResources(c.WannaDanceResources) {
			return fmt.Errorf("invalid WannaDance resources: %s", c.WannaDanceResources)
		}
		if !ValidRoomResources(c.DuDuFitDanceResources) {
			return fmt.Errorf("invalid DuDuFitDance resources: %s", c.DuDuFitDanceResources)
		}
		return nil
	}

	switch field {
	case keyYtMode:
		if !ValidMode(c.YoutubeMode) {
			return fmt.Errorf("invalid mode: %s", c.YoutubeMode)
		}
	case keyBiliMode:
		if !ValidMode(c.BiliBiliMode) {
			return fmt.Errorf("invalid mode: %s", c.BiliBiliMode)
		}
	case keyYtAllowedResources:
		if !ValidPlatformResources(c.YoutubeResources) {
			return fmt.Errorf("invalid resources: %s", c.YoutubeResources)
		}
	case keyBiliAllowedResources:
		if !ValidPlatformResources(c.BiliBiliResources) {
			return fmt.Errorf("invalid resources: %s", c.BiliBiliResources)
		}
	case keyPypyAllowedResources:
		if !ValidRoomResources(c.PyPyDanceResources) {
			return fmt.Errorf("invalid resources: %s", c.PyPyDanceResources)
		}
	case keyWannaAllowedResources:
		if !ValidRoomResources(c.WannaDanceResources) {
			return fmt.Errorf("invalid resources: %s", c.WannaDanceResources)
		}
	case keyDuduAllowedResources:
		if !ValidRoomResources(c.DuDuFitDanceResources) {
			return fmt.Errorf("invalid resources: %s", c.DuDuFitDanceResources)
		}
	}
	return nil
}
