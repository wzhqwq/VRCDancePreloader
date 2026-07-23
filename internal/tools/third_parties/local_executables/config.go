package local_executables

import (
	"fmt"
	"os/exec"
)

type Config struct {
	CheckUpdateOnStart bool `yaml:"check-update-on-start"`

	YtDlpPath string `yaml:"yt-dlp-path"`
	DenoPath  string `yaml:"deno-path"`
}

func DefaultConfig() Config {
	return Config{
		YtDlpPath: "<vrcdp>",
		DenoPath:  "<vrcdp>",
	}
}

func (c Config) Validate(field string) error {
	if field == "" {
		if !validPath(c.YtDlpPath) {
			return fmt.Errorf("invalid path of yt-dlp: %s", c.YtDlpPath)
		}
		if !validPath(c.DenoPath) {
			return fmt.Errorf("invalid path of deno: %s", c.DenoPath)
		}
		return nil
	}

	switch field {
	case "yt-dlp-path":
		if !validPath(c.YtDlpPath) {
			return fmt.Errorf("invalid path of yt-dlp: %s", c.YtDlpPath)
		}
	case "deno-path":
		if !validPath(c.DenoPath) {
			return fmt.Errorf("invalid path of deno: %s", c.DenoPath)
		}
	}

	return nil
}

func validPath(path string) bool {
	if path == "" || path == "<vrcdp>" {
		return true
	}
	if _, err := exec.LookPath(path); err != nil {
		return false
	}
	return true
}
