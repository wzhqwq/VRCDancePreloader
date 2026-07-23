package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"gopkg.in/yaml.v3"
)

var logger = utils.NewLogger("Config File")

func LoadConfigV3(cfg *Config) bool {
	var configPath = filepath.Join(custom_fyne.AppConfigRoot, "app.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false
	}
	_, err := os.Stat(configPath)
	if errors.Is(err, os.ErrPermission) {
		logger.FatalLn("config.yaml permission denied")
	}

	if err == nil {
		configFile, err := os.Open(configPath)
		if err != nil {
			logger.FatalLnf("Failed to open config.yaml: %s", err)
			logger.WarnLn("Use default config instead.")
		}
		defer configFile.Close()

		decoder := yaml.NewDecoder(configFile)
		err = decoder.Decode(&cfg)
		if err != nil {
			logger.FatalLnf("Failed to parse config.yaml: %s", err)
			logger.WarnLn("Use default config instead.")
		}
	}

	return true
}

func SaveFileAtomic(cfg any) error {
	var configPath = filepath.Join(custom_fyne.AppConfigRoot, "app.yaml")

	if err := os.MkdirAll(custom_fyne.AppConfigRoot, os.ModePerm); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(custom_fyne.AppConfigRoot, ".app.yaml.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp config file: %w", err)
	}

	tmpPath := tmpFile.Name()
	committed := false

	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	mode := os.FileMode(0644)
	if info, statErr := os.Stat(configPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err = tmpFile.Chmod(mode); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp config file: %w", err)
	}

	encoder := yaml.NewEncoder(tmpFile)
	if err = encoder.Encode(cfg); err != nil {
		_ = encoder.Close()
		_ = tmpFile.Close()
		return fmt.Errorf("encode config yaml: %w", err)
	}

	if err = encoder.Close(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("close yaml encoder: %w", err)
	}

	if err = tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp config file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config file: %w", err)
	}

	if err = os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("replace config file: %w", err)
	}

	committed = true

	if dirFile, err := os.Open(custom_fyne.AppConfigRoot); err == nil {
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}

	return nil
}
