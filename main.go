package main

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/download"
	"log"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/PyPyDancePreloader/config"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/proxy"
	"github.com/wzhqwq/PyPyDancePreloader/internal/requesting"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song_ui/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song_ui/tui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/PyPyDancePreloader/internal/watcher"

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

func processKeyConfig() {
	keyConfig := config.GetKeyConfig()
	if keyConfig.Youtube != "" {
		third_party_api.SetYoutubeApiKey(keyConfig.Youtube)
	} else {
		log.Println("[Warning] Youtube API key not set, so the title of Youtube songs might not display correctly")
	}
}

func processProxyConfig() {
	proxyConfig := config.GetProxyConfig()
	keyConfig := config.GetKeyConfig()
	requesting.InitPypyClient(proxyConfig.Pypy)
	requesting.InitYoutubeVideoClient(proxyConfig.YoutubeVideo)
	requesting.InitYoutubeImageClient(proxyConfig.YoutubeImage)
	if keyConfig.Youtube != "" {
		requesting.InitYoutubeApiClient(proxyConfig.YoutubeApi)
	}
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
	processKeyConfig()
	processProxyConfig()
	limits := config.GetLimitConfig()

	// Listen for interrupt
	osSignalCh := make(chan os.Signal, 1)
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	// The ending note
	defer log.Println("Gracefully stopped")

	i18n.Init()

	err := cache.InitSongList()
	if err != nil {
		log.Println("Failed to init cache:", err)
		return
	}

	cache.SetupCache("./cache", limits.MaxCache*1024*1024)
	defer func() {
		log.Println("Cleaning up cache")
		cache.CleanUpCache()
	}()

	download.InitDownloadManager(limits.MaxDownload)
	defer func() {
		log.Println("Stopping all downloading tasks")
		download.StopAllAndWait()
	}()

	select {
	case <-osSignalCh:
		return
	default:
	}
	playlist.Init(limits.MaxPreload)
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
