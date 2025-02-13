package main

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/proxy"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/tui"
	"github.com/wzhqwq/VRCDancePreloader/internal/watcher"

	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var build_gui_on = false

var args struct {
	Port string `arg:"-p,--port" default:"7653" help:"port to listen on"`

	VrChatDir string `arg:"-d,--vrchat-dir" default:"" help:"VRChat directory"`

	GuiEnabled bool `arg:"-g,--gui" default:"false" help:"enable GUI"`
	TuiEnabled bool `arg:"-t,--tui" default:"false" help:"enable TUI"`

	SkipClientTest bool `arg:"--skip-client-test" default:"false" help:"skip client connectivity test"`

	// experimental

	AsyncDownload bool `arg:"-a,--async-download" default:"false" help:"experimental, allow preloader to respond partial data during downloading, which is useful in random play"`
}

func main() {
	arg.MustParse(&args)

	// Apply build tag
	if build_gui_on {
		args.GuiEnabled = true
	}

	// Apply argument config
	requesting.SetSkipTest(args.SkipClientTest)
	playlist.SetAsyncDownload(args.AsyncDownload)

	// Apply config.yaml
	config.LoadConfig()
	config.GetKeyConfig().Init()
	config.GetProxyConfig().Init()

	// Listen for interrupt
	osSignalCh := make(chan os.Signal, 1)
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	// The ending note
	defer log.Println("Gracefully stopped")

	i18n.Init()

	err := cache.InitSongList()
	if err != nil {
		log.Println("Failed to fetch pypy song list:", err)
		return
	}

	config.GetCacheConfig().Init()
	defer func() {
		log.Println("Cleaning up cache")
		cache.CleanUpCache()
	}()

	config.GetDownloadConfig().Init()
	defer func() {
		log.Println("Stopping all downloading tasks")
		download.StopAllAndWait()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	config.GetPreloadConfig().Init()
	defer func() {
		log.Println("Stopping playlist")
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
			log.Println("Failed to get user config directory:", err)
			return
		}
		logDir = filepath.Join(roaming, "..", "LocalLow", "VRChat", "VRChat")
	}
	err = watcher.Start(logDir)
	if err != nil {
		log.Println("Failed to start watcher:", err)
		return
	}
	defer func() {
		log.Println("Stopping log watcher")
		watcher.Stop()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	go proxy.Start(args.Port)
	defer func() {
		log.Println("Stopping proxy")
		proxy.Stop()
	}()

	if args.TuiEnabled {
		select {
		case <-osSignalCh:
			return
		default:
		}
		tui.Start()
		defer func() {
			log.Println("Stopping TUI")
			tui.Stop()
		}()
	}

	if args.GuiEnabled {
		select {
		case <-osSignalCh:
			return
		default:
		}
		gui.Start()
		defer func() {
			log.Println("Stopping GUI")
			gui.Stop()
		}()

		go func() {
			<-osSignalCh
			log.Println("Quitting...")
			window_app.Quit()
		}()
		window_app.MainLoop()
	} else {
		<-osSignalCh
	}
}
