package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
)

type HijackConfig struct {
	ProxyPort        int      `yaml:"proxy-port"`
	InterceptedSites []string `yaml:"intercepted-sites"`
	EnableHttps      bool     `yaml:"enable-https"`
	EnablePWI        bool     `yaml:"enable-pwi"`
	LimitBandwidth   bool     `yaml:"limit-bandwidth"`
	MaxBandwidth     int      `yaml:"max-bandwidth"`

	HijackRunner *input.ServerRunner `yaml:"-"`
}

func GetHijackConfig() *HijackConfig {
	return &config.Hijack
}

var defaultHijackConfig = HijackConfig{
	ProxyPort:        7653,
	InterceptedSites: constants.CopyAllSites(),
	MaxBandwidth:     25,
}

func (hc *HijackConfig) Init() {
}

func (hc *HijackConfig) Stop() {
}

func (hc *HijackConfig) UpdatePort(port int) {
}

func (hc *HijackConfig) UpdateEnableHttps(b bool) {
}

func (hc *HijackConfig) UpdateSites(sites []string) {
}

func (hc *HijackConfig) UpdateEnablePWI(b bool) {
}

func (hc *HijackConfig) UpdateLimitBandwidth(b bool) {
}

func (hc *HijackConfig) UpdateMaxBandwidth(b int) {
}
