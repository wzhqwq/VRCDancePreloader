package config

import "github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"

type YoutubeConfig struct {
	EnableApi       bool `yaml:"enable-youtube-api"`
	EnableThumbnail bool `yaml:"enable-youtube-thumbnail"`
	EnableYtDlp     bool `yaml:"enable-ytdlp"`
}

func GetYoutubeConfig() *YoutubeConfig {
	return &config.Youtube
}

var defaultYoutubeConfig = YoutubeConfig{}

func (yc *YoutubeConfig) Init() {
	third_party_api.EnableYoutubeApi = yc.EnableApi
	third_party_api.EnableYoutubeThumbnail = yc.EnableThumbnail
	third_party_api.EnableYtDlp = yc.EnableYtDlp
}

func (yc *YoutubeConfig) UpdateEnableApi(enabled bool) {
	yc.EnableApi = enabled
	saveAndNotify("youtube")

	third_party_api.EnableYoutubeApi = enabled
}

func (yc *YoutubeConfig) UpdateEnableThumbnail(enabled bool) {
	yc.EnableThumbnail = enabled
	saveAndNotify("youtube")

	third_party_api.EnableYoutubeThumbnail = enabled
}

func (yc *YoutubeConfig) UpdateEnableYtDlp(enabled bool) {
	yc.EnableYtDlp = enabled
	saveAndNotify("youtube")

	third_party_api.EnableYtDlp = enabled
}
