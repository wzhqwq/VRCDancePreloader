package requesting

import (
	"fmt"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

var cfg Config

type Tool struct {
	service.ConfigurableTool
}

func initialize() error {
	initClient(PyPyDance, cfg.Pypy, true)
	initClient(WannaDance, cfg.Wanna, true)
	initClient(DuDuFitDance, cfg.DuDu, true)
	initClient(BiliBili, cfg.BiliBili, true)
	initClient(YouTubeVideo, cfg.YoutubeVideo, true)
	initClient(YouTubeImage, cfg.YoutubeImage, true)
	initClient(YouTubeApi, cfg.YoutubeApi, true)
	initClient(GitHubApi, cfg.GitHubApi, true)
	initClient(GitHubAssets, cfg.GitHubAssets, true)

	return nil
}

func destroy() error {
	shutdownClients()
	return nil
}

func New(c Config) *Tool {
	cfg = c

	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool(initialize, destroy),
	}
}

func UpdateConfig(c Config, field string) error {
	if field == "" {
		updateIfNotTheSame(PyPyDance, cfg.Pypy, c.Pypy)
		updateIfNotTheSame(WannaDance, cfg.Wanna, c.Wanna)
		updateIfNotTheSame(DuDuFitDance, cfg.DuDu, c.DuDu)
		updateIfNotTheSame(BiliBili, cfg.BiliBili, c.BiliBili)
		updateIfNotTheSame(YouTubeVideo, cfg.YoutubeVideo, c.YoutubeVideo)
		updateIfNotTheSame(YouTubeApi, cfg.YoutubeApi, c.YoutubeApi)
		updateIfNotTheSame(YouTubeImage, cfg.YoutubeImage, c.YoutubeImage)
		updateIfNotTheSame(GitHubApi, cfg.GitHubApi, c.GitHubApi)
		updateIfNotTheSame(GitHubAssets, cfg.GitHubAssets, c.GitHubAssets)
	} else {
		switch field {
		case "pypydance-api":
			updateIfNotTheSame(PyPyDance, cfg.Pypy, c.Pypy)
		case "wannadance-api":
			updateIfNotTheSame(WannaDance, cfg.Wanna, c.Wanna)
		case "dudu-fitdance-api":
			updateIfNotTheSame(DuDuFitDance, cfg.DuDu, c.DuDu)
		case "bilibili":
			updateIfNotTheSame(BiliBili, cfg.BiliBili, c.BiliBili)
		case "youtube-video":
			updateIfNotTheSame(YouTubeVideo, cfg.YoutubeVideo, c.YoutubeVideo)
		case "youtube-api":
			updateIfNotTheSame(YouTubeApi, cfg.YoutubeApi, c.YoutubeApi)
		case "youtube-image":
			updateIfNotTheSame(YouTubeImage, cfg.YoutubeImage, c.YoutubeImage)
		case "github-api":
			updateIfNotTheSame(GitHubApi, cfg.GitHubApi, c.GitHubApi)
		case "github-assets":
			updateIfNotTheSame(GitHubAssets, cfg.GitHubAssets, c.GitHubAssets)
		default:
			return fmt.Errorf("unknown proxy item: %s", field)
		}
	}

	cfg = c
	return nil
}

func updateIfNotTheSame(name ClientName, old, new string) {
	if old != new {
		updateClient(name, new)
	}
}
