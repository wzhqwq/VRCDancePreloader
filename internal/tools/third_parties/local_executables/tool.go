package local_executables

import (
	"sync/atomic"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var cfg Config

// stopCh cancels the background work of this tool.
//
// A ConfigurableTool is started and stopped exactly once in the lifetime of the
// process (it exposes no restart: see service.ConfigurableTool), so destroy closes
// this without any guard. Should a restart path ever be added, this close is where
// it would show up — as a panic on a closed channel, not as a silent half start.
var stopCh = make(chan struct{})

var ytdlpAvailableEm = utils.NewEventManager[bool]()

// hasYtDlp is written by the pump below and read by YtDlpAvailable from arbitrary
// goroutines (the providers' loops, the yt-dlp resolver), so it is atomic.
var hasYtDlp atomic.Bool

func initialize() error {
	InitDeno()
	InitYtDlp()

	ytdlpVerCh := Get("ytdlp").Subscribe()
	denoVerCh := Get("deno").Subscribe()

	hasYtDlp.Store(Get("ytdlp").Info().Version != "")

	go func() {
		defer ytdlpVerCh.Close()
		defer denoVerCh.Close()

		for {
			select {
			case <-stopCh:
				return
			case <-ytdlpVerCh.Channel:
				// The channel delivers the *kind* of change (BinVersion, BinState,
				// BinProgress), not a version string: testing it for "" made every
				// event report yt-dlp as available. The version itself is what the
				// answer depends on, and the event is what orders the read after the
				// write that produced it.
				available := Get("ytdlp").Info().Version != ""
				hasYtDlp.Store(available)
				ytdlpAvailableEm.NotifySubscribers(available)
			case <-denoVerCh.Channel:
				if Get("deno").Info().Version != "" && hasYtDlp.Load() {
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
		ConfigurableTool: service.ConstructConfigurableTool("local_executables", initialize, destroy, logger),
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
	return hasYtDlp.Load()
}

func updateIfNotTheSame(initializer func(), old, new string) {
	if old != new {
		initializer()
	}
}
