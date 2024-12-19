package main

import (
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

var args struct {
	Port       string `arg:"-p,--port" default:"7653" help:"port to listen on"`
	VrChatDir  string `arg:"-d,--vrchat-dir" default:"" help:"VRChat directory"`
	GuiEnabled bool   `arg:"-g,--gui" default:"false" help:"enable GUI"`
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
	if keyConfig.Youtube != "" {
		requesting.InitYoutubeApiClient(proxyConfig.YoutubeApi)
	}
}

func main() {
	arg.MustParse(&args)
	config.LoadConfig()
	processKeyConfig()
	processProxyConfig()
	limits := config.GetLimitConfig()

	osSignalCh := make(chan os.Signal, 1)

	// listen for SIGINT and SIGTERM
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-osSignalCh

		log.Println("Stopping proxy")
		proxy.Stop()
		log.Println("Stopping watcher")
		watcher.Stop()
		log.Println("Stopping playlist")
		playlist.StopPlayList()
		log.Println("Stopping tui")
		tui.Stop()
		log.Println("Stopping cache")
		cache.CleanUpCache()
		log.Println("Gracefully stopped")
		os.Exit(0)
	}()

	i18n.Init()

	err := cache.InitCache("./cache", limits.MaxCache*1024*1024, limits.MaxDownload)
	if err != nil {
		log.Println("Failed to init cache:", err)
		return
	}
	defer cache.CleanUpCache()

	playlist.Init(limits.MaxPreload)
	defer playlist.StopPlayList()

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
	defer watcher.Stop()

	go proxy.Start(args.Port)
	defer proxy.Stop()

	tui.Start()
	defer tui.Stop()

	if args.GuiEnabled {
		gui.Start()
		defer gui.Stop()

		window_app.MainLoop()
	} else {
		for {
			select {}
		}
	}
}
