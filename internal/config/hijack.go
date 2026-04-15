package config

import (
	"strconv"

	"github.com/wzhqwq/VRCDancePreloader/internal/constants"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/input"
	"github.com/wzhqwq/VRCDancePreloader/internal/hijack"
	"github.com/wzhqwq/VRCDancePreloader/internal/service"
)

type HijackConfig struct {
	ProxyPort        int      `yaml:"proxy-port"`
	InterceptedSites []string `yaml:"intercepted-sites"`
	EnableHttps      bool     `yaml:"enable-https"`
	EnablePWI        bool     `yaml:"enable-pwi"`
	LimitBandwidth   bool     `yaml:"limit-bandwidth"`

	HijackRunner *input.ServerRunner `yaml:"-"`
}

func GetHijackConfig() *HijackConfig {
	return &config.Hijack
}

var defaultHijackConfig = HijackConfig{
	ProxyPort:        7653,
	InterceptedSites: constants.CopyAllSites(),
}

func (hc *HijackConfig) Init() {
	runner := input.NewServerRunner(hc.ProxyPort)
	runner.OnSave = hc.UpdatePort
	runner.StartServer = func() error {
		if err := hijack.Start(hc.InterceptedSites, hc.EnableHttps, hc.ProxyPort); err != nil {
			if global_state.IsInGui() {
				return err
			}

			logger.FatalLn("Failed to start hijack server:", err)
		}
		return nil
	}
	runner.StopServer = config.Hijack.Stop
	runner.Run()

	hc.HijackRunner = runner
	if hc.EnablePWI {
		service.StartPWIServer()
	}
	//service.StartStubServer()
	service.ProxyServerPort = strconv.Itoa(hc.ProxyPort)
}

func (hc *HijackConfig) Stop() {
	hijack.Stop()
	if hc.EnablePWI {
		service.StopPWIServer()
	}
	//service.StopStubServer()
}

func (hc *HijackConfig) UpdatePort(port int) {
	hc.ProxyPort = port
	SaveConfig()
	service.ProxyServerPort = strconv.Itoa(hc.ProxyPort)
}

func (hc *HijackConfig) UpdateEnableHttps(b bool) {
	hc.EnableHttps = b
	hc.HijackRunner.Run()
	SaveConfig()
}

func (hc *HijackConfig) UpdateSites(sites []string) {
	hc.InterceptedSites = sites
	hc.HijackRunner.Run()
	SaveConfig()
}

func (hc *HijackConfig) UpdateEnablePWI(b bool) {
	hc.EnablePWI = b
	if hc.EnablePWI {
		service.StartPWIServer()
	} else {
		service.StopPWIServer()
	}
	SaveConfig()
}

func (hc *HijackConfig) UpdateLimitBandwidth(b bool) {
	hc.LimitBandwidth = b
	hijack.SetLimitBandwidth(b)
	SaveConfig()
}
