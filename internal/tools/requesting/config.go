package requesting

import (
	"fmt"
	"net/url"
)

type Config struct {
	Pypy  string `yaml:"pypydance-api"`
	Wanna string `yaml:"wannadance-api"`
	DuDu  string `yaml:"dudu-fitdance-api"`

	BiliBili string `yaml:"bilibili"`

	YoutubeVideo string `yaml:"youtube-video"`
	YoutubeApi   string `yaml:"youtube-api"`
	YoutubeImage string `yaml:"youtube-image"`

	GitHubApi    string `yaml:"github-api"`
	GitHubAssets string `yaml:"github-assets"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c Config) Validate(field string) error {
	if field == "" {
		return batchValidate(
			c.Pypy, c.Wanna, c.DuDu,
			c.BiliBili,
			c.YoutubeVideo, c.YoutubeApi, c.YoutubeImage,
			c.GitHubApi, c.GitHubAssets,
		)
	}
	return batchValidate(c.Value(field))
}

func (c Config) Value(field string) string {
	switch field {
	case "pypydance-api":
		return c.Pypy
	case "wannadance-api":
		return c.Wanna
	case "dudu-fitdance-api":
		return c.DuDu
	case "bilibili":
		return c.BiliBili
	case "youtube-video":
		return c.YoutubeVideo
	case "youtube-api":
		return c.YoutubeApi
	case "youtube-image":
		return c.YoutubeImage
	case "github-api":
		return c.GitHubApi
	case "github-assets":
		return c.GitHubAssets
	default:
		return ""
	}
}

func (c Config) Set(field, value string) (Config, error) {
	switch field {
	case "pypydance-api":
		c.Pypy = value
	case "wannadance-api":
		c.Wanna = value
	case "dudu-fitdance-api":
		c.DuDu = value
	case "bilibili":
		c.BiliBili = value
	case "youtube-video":
		c.YoutubeVideo = value
	case "youtube-api":
		c.YoutubeApi = value
	case "youtube-image":
		c.YoutubeImage = value
	case "github-api":
		c.GitHubApi = value
	case "github-assets":
		c.GitHubAssets = value
	default:
		return c, fmt.Errorf("unknown field: %s", field)
	}
	return c, nil
}

func batchValidate(urls ...string) error {
	for _, value := range urls {
		if value == "<SYSTEM>" {
			continue
		}
		_, err := url.Parse(value)
		if err != nil {
			return err
		}
	}
	return nil
}
