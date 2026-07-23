package local_executables

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cfg Config
var stopCh = make(chan struct{})

var ytdlpAvailableEm = utils.NewEventManager[bool]()
var hasYtDlp = false

func initialize() error {
	InitDeno()
	InitYtDlp()

	ytdlpVerCh := Get("ytdlp").Subscribe()
	denoVerCh := Get("deno").Subscribe()

	hasYtDlp = Get("ytdlp").Info.Version != ""

	go func() {
		defer ytdlpVerCh.Close()
		defer denoVerCh.Close()

		for {
			select {
			case <-stopCh:
				return
			case ver := <-ytdlpVerCh.Channel:
				hasYtDlp = ver != ""
				ytdlpAvailableEm.NotifySubscribers(hasYtDlp)
			case ver := <-denoVerCh.Channel:
				if ver != "" && hasYtDlp {
					// retry ytdlp when deno becomes available
					ytdlpAvailableEm.NotifySubscribers(true)
				}
			}
		}
	}()

	return nil
}

func destroy() error {
	downloadableMap["ytdlp"].Stop()
	downloadableMap["deno"].Stop()
	close(stopCh)
	return nil
}

type Tool struct {
	service.ConfigurableTool
}

func New(c Config) *Tool {
	cfg = c

	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool(initialize, destroy),
	}
}

func UpdateConfig(c Config, field string) error {
	if field == "" {
		updateIfNotTheSame(InitYtDlp, cfg.YtDlpPath, c.YtDlpPath)
		updateIfNotTheSame(InitDeno, cfg.DenoPath, c.DenoPath)
	} else {
		switch field {
		case "yt-dlp-path":
			updateIfNotTheSame(InitYtDlp, cfg.YtDlpPath, c.YtDlpPath)
		case "deno-path":
			updateIfNotTheSame(InitDeno, cfg.DenoPath, c.DenoPath)
		}
	}

	cfg = c
	return nil
}

func SubscribeYtDlpAvailability() *utils.EventSubscriber[bool] {
	return ytdlpAvailableEm.SubscribeEvent()
}

func YtDlpAvailable() bool {
	return hasYtDlp
}

func updateIfNotTheSame(initializer func(), old, new string) {
	if old != new {
		initializer()
	}
}
