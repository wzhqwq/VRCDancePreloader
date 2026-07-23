package cache_manager

import "errors"

type Config struct {
	Path          string `yaml:"path"`
	MaxVideoCache int    `yaml:"max-video-cache"`
	KeepFavorites bool   `yaml:"keep-favorites"`
	//RWBufferSize  int    `yaml:"rw-buffer-size"`
	// 0: legacy, 1: continuous, 2: fragmented
	VideoFileFormat int `yaml:"video-file-format"`

	ForceExpiration bool `yaml:"force-expiration"`
}

func (c Config) Validate(field string) error {
	if (field == "" || field == "video-file-format") && (c.VideoFileFormat <= 0 || c.VideoFileFormat > 2) {
		return errors.New("video-file-format must be 1 (continuous) or 2 (fragmented)")
	}
	if (field == "" || field == "max-video-cache") && c.MaxVideoCache <= 0 {
		return errors.New("max-video-cache must be greater than zero")
	}
	return nil
}

func DefaultConfig() Config {
	return Config{
		Path:          "./cache",
		MaxVideoCache: 300,
		//RWBufferSize:  1,
		VideoFileFormat: 1,
	}
}
