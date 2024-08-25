package main

import (
	"log"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
	"github.com/wzhqwq/PyPyDancePreloader/internal/proxy"
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
}

func main() {
	arg.MustParse(&args)

	osSignalCh := make(chan os.Signal, 1)

	// listen for SIGINT and SIGTERM
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-osSignalCh

		proxy.Stop()
		watcher.Stop()
		playlist.Stop()
		cache.CleanUpCache()
		log.Println("Gracefully stopped")
		os.Exit(0)
	}()

	i18n.Init()
	cache.InitCache("./cache", args.CacheDiskMax)
	playlist.Init(args.PreloadMax, args.DownloadMax)

	logDir := args.VrChatDir
	if logDir == "" {
		roaming, err := os.UserConfigDir()
		if err != nil {
			log.Println("Failed to get user config directory:", err)
			return
		}
		logDir = filepath.Join(roaming, "..", "LocalLow", "VRChat", "VRChat")
	}
	watcher.Start(logDir)

	go proxy.Start(args.Port)
	if args.GuiEnabled {
		gui.InitGui()
		gui.MainLoop(osSignalCh)
	}
	for {
		select {}
	}
}
