package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/live"
)

type LiveConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Port     int    `yaml:"port"`
	Settings string `yaml:"settings"`

	LiveRunner *input.ServerRunner `yaml:"-"`
}

func GetLiveConfig() *LiveConfig {
	return &config.Live
}

var defaultLiveConfig = LiveConfig{
	Port:     7652,
	Settings: "{}",
}

func (lc *LiveConfig) Init() {
	live.OnSettingsChanged = func(settings string) {
		lc.UpdateSettings(settings)
	}
	live.GetSettings = func() string {
		return lc.Settings
	}

	runner := input.NewServerRunner(lc.Port)
	runner.OnSave = lc.UpdatePort
	runner.StartServer = func() error {
		if err := live.StartLiveServer(lc.Port); err != nil {
			if global_state.IsInGui() {
				return err
			}

			logger.FatalLn("Failed to start live server:", err)
		}
		return nil
	}
	runner.StopServer = config.Hijack.Stop
	lc.LiveRunner = runner

	if lc.Enabled {
		runner.Run()
	}
}

func (lc *LiveConfig) UpdateEnable(b bool) {
	lc.Enabled = b
	if lc.Enabled {
		lc.LiveRunner.Run()
	} else {
		live.StopLiveServer()
	}
	SaveConfig()
}

func (lc *LiveConfig) UpdatePort(port int) {
	lc.Port = port
	SaveConfig()
}

func (lc *LiveConfig) UpdateSettings(settings string) {
	lc.Settings = settings
	SaveConfig()
}

func (lc *LiveConfig) Stop() {
	live.StopLiveServer()
}
