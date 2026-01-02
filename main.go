package main

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tui"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher"

	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var buildGuiOn = false

var logger = utils.NewLogger("VRCDP")

var args struct {
	VrChatDir string `arg:"-d,--vrchat-dir" default:"" help:"VRChat directory"`

	GuiEnabled bool `arg:"-g,--gui" default:"false" help:"enable GUI"`
	TuiEnabled bool `arg:"-t,--tui" default:"false" help:"enable TUI"`

	SkipClientTest bool `arg:"--skip-client-test" default:"false" help:"skip client connectivity test"`

	// switches

	DisableAsyncDownload bool `arg:"--disable-async-download" default:"false" help:"disable async download"`
}

func main() {
	arg.MustParse(&args)

	// Apply build tag
	if buildGuiOn {
		args.GuiEnabled = true
	}

	// gui state
	if args.GuiEnabled {
		global_state.RunInGui()
	}

	// Apply argument config
	config.SetSkipTest(args.SkipClientTest)
	if args.DisableAsyncDownload {
		playlist.SetAsyncDownload(false)
	}

	i18n.Init()

	// Apply config.yaml
	config.LoadConfig()
	config.GetYoutubeConfig().Init()
	config.GetKeyConfig().Init()
	config.GetProxyConfig().Init()

	// Listen for interrupt
	osSignalCh := make(chan os.Signal, 1)
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	// The ending note
	defer logger.InfoLn("Gracefully stopped")

	songListCtx, cancel := context.WithCancel(context.Background())
	cache.InitSongList(songListCtx)
	defer cancel()

	err := config.GetDbConfig().Init()
	if err != nil {
		logger.InfoLn("Failed to init database:", err)
		return
	}
	defer func() {
		logger.InfoLn("Closing database")
		persistence.CloseDB()
	}()

	config.GetCacheConfig().Init()
	defer func() {
		logger.InfoLn("Stopping cache")
		cache.StopCache()
	}()

	config.GetDownloadConfig().Init()
	defer func() {
		logger.InfoLn("Stopping all downloading tasks")
		download.StopAllAndWait()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	config.GetPreloadConfig().Init()
	defer func() {
		logger.InfoLn("Stopping playlist")
		playlist.StopPlayList()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	logDir := args.VrChatDir
	if logDir == "" {
		roaming, err := os.UserConfigDir()
		if err != nil {
			logger.ErrorLn("Failed to get user config directory:", err)
			return
		}
		logDir = filepath.Join(roaming, "..", "LocalLow", "VRChat", "VRChat")
	}
	err = watcher.Start(logDir)
	if err != nil {
		logger.ErrorLn("Failed to start watcher:", err)
		return
	}
	defer func() {
		logger.InfoLn("Stopping log watcher")
		watcher.Stop()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	config.GetHijackConfig().Init()
	defer func() {
		logger.InfoLn("Stopping proxy")
		config.GetHijackConfig().Stop()
	}()

	config.GetLiveConfig().Init()
	defer func() {
		logger.InfoLn("Stopping live")
		config.GetLiveConfig().Stop()
	}()

	if args.TuiEnabled {
		select {
		case <-osSignalCh:
			return
		default:
		}
		tui.Start()
		defer func() {
			logger.InfoLn("Stopping TUI")
			tui.Stop()
		}()
	} else if args.GuiEnabled {
		select {
		case <-osSignalCh:
			return
		default:
		}
		main_window.Start()
		defer func() {
			logger.InfoLn("Stopping GUI")
			main_window.Stop()
		}()

		go func() {
			<-osSignalCh
			logger.InfoLn("Quitting...")
			custom_fyne.Quit()
		}()
		custom_fyne.MainLoop()
	} else {
		<-osSignalCh
	}
}
