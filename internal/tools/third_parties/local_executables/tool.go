package local_executables

import (
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

// ytdlpAvailable is the level every consumer reads: it is a utils.Level so that
// a reader re-reads it when it is woken instead of trusting the wake-up, and so
// that the value is always stored before the wake-up is sent.
var ytdlpAvailable = utils.NewLevel(false)

func initialize() error {
	InitDeno()
	InitYtDlp()

	ytdlpVerCh := Get("ytdlp").Subscribe()
	denoVerCh := Get("deno").Subscribe()

	ytdlpAvailable.Store(Get("ytdlp").Info().Version != "")

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
				ytdlpAvailable.Store(Get("ytdlp").Info().Version != "")
			case <-denoVerCh.Channel:
				if Get("deno").Info().Version != "" && ytdlpAvailable.Current() {
					// retry ytdlp when deno becomes available
					ytdlpAvailable.Wake()
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

// YtDlpAvailability hands a consumer its own watcher of the availability level:
// wait on Wakes, then read YtDlpAvailable (or the watcher's Current). The caller
// closes it, so a provider loop that exits stops being woken.
func YtDlpAvailability() *utils.LevelWatcher[bool] {
	return ytdlpAvailable.Subscribe()
}

func YtDlpAvailable() bool {
	return ytdlpAvailable.Current()
}

func updateIfNotTheSame(initializer func(), old, new string) {
	if old != new {
		initializer()
	}
}
