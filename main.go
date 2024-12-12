package main

import (
	"log"
	"regexp"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/proxy"
	"github.com/wzhqwq/PyPyDancePreloader/internal/song_ui/tui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/watcher"

	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var args struct {
	Port         string `arg:"-p,--port" default:"7653" help:"port to listen on"`
	CacheDiskMax int    `arg:"-c,--cache" default:"100" help:"maximum disk cache size(MB)"`
	VrChatDir    string `arg:"-d,--vrchat-dir" default:"" help:"VRChat directory"`
	GuiEnabled   bool   `arg:"-g,--gui" default:"true" help:"enable GUI"`
	PreloadMax   int    `arg:"--max-preload" default:"4" help:"maximum preload count"`
	DownloadMax  int    `arg:"--max-download" default:"2" help:"maximum parallel download count"`
	Proxy        string `arg:"--proxy" default:"" help:"proxy server, example: 127.0.0.1:7890, set for both http and https"`
}

func main() {
	arg.MustParse(&args)

	if args.Proxy != "" {
		// check format
		if regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}:\d+$`).MatchString(args.Proxy) {
			os.Setenv("HTTP_PROXY", "http://"+args.Proxy)
			os.Setenv("HTTPS_PROXY", "http://"+args.Proxy)
		} else {
			log.Println("Invalid proxy format, should be like host:port")
			os.Exit(1)
		}
	}

	osSignalCh := make(chan os.Signal, 1)

	// listen for SIGINT and SIGTERM
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-osSignalCh

		proxy.Stop()
		watcher.Stop()
		playlist.StopPlayList()
		tui.Stop()
		cache.CleanUpCache()
		log.Println("Gracefully stopped")
		os.Exit(0)
	}()

	i18n.Init()
	cache.InitCache("./cache", args.CacheDiskMax*1024*1024, args.DownloadMax)
	playlist.Init(args.PreloadMax)

	logDir := args.VrChatDir
	if logDir == "" {
		roaming, err := os.UserConfigDir()
		if err != nil {
			log.Println("Failed to get user config directory:", err)
			return
		}
		logDir = filepath.Join(roaming, "..", "LocalLow", "VRChat", "VRChat")
	}
	err := watcher.Start(logDir)
	if err != nil {
		log.Println("Failed to start watcher:", err)
		return
	}

	go proxy.Start(args.Port)
	tui.Start()
	if args.GuiEnabled {
		gui.InitGui()
		gui.MainLoop(osSignalCh)
	}
	for {
		select {}
	}
}
