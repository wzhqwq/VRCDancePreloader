package main

import (
	"log"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/PyPyDancePreloader/internal/cache"
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
}

func main() {
	arg.MustParse(&args)

	go func() {
		// listen for SIGINT and SIGTERM
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c

		proxy.Stop()
		watcher.Stop()
		playlist.Stop()
		cache.CleanUpCache()
		log.Println("Gracefully stopped")
		os.Exit(0)
	}()

	cache.InitCache("./cache", args.CacheDiskMax)
	playlist.Init()

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

	proxy.Start(args.Port)
}
