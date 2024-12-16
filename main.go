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
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
	"github.com/wzhqwq/PyPyDancePreloader/internal/watcher"

	"gopkg.in/yaml.v3"

	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var args struct {
	Port         string `arg:"-p,--port" default:"7653" help:"port to listen on"`
	CacheDiskMax int    `arg:"-c,--cache" default:"100" help:"maximum disk cache size(MB)"`
	VrChatDir    string `arg:"-d,--vrchat-dir" default:"" help:"VRChat directory"`
	GuiEnabled   bool   `arg:"-g,--gui" default:"false" help:"enable GUI"`
	PreloadMax   int    `arg:"--max-preload" default:"4" help:"maximum preload count"`
	DownloadMax  int    `arg:"--max-download" default:"2" help:"maximum parallel download count"`
	Proxy        string `arg:"--proxy" default:"" help:"proxy server, example: 127.0.0.1:7890, set for both http and https"`
}

var keyConfig struct {
	Youtube string `yaml:"youtube"`
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

	// load key config (if exists)
	keyConfigFile, err := os.Open("keys.yaml")
	if err == nil {
		decoder := yaml.NewDecoder(keyConfigFile)
		err = decoder.Decode(&keyConfig)
		if err != nil {
			log.Println("Failed to load key config:", err)
		}
		keyConfigFile.Close()
	}
	if keyConfig.Youtube != "" {
		utils.SetYoutubeApiKey(keyConfig.Youtube)
	} else {
		log.Println("[Warning] Youtube API key not set, so the title of Youtube songs might not display correctly")
	}

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
	cache.InitCache("./cache", args.CacheDiskMax*1024*1024, args.DownloadMax)
	defer cache.CleanUpCache()

	playlist.Init(args.PreloadMax)
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
		gui.InitGui()
		gui.MainLoop(osSignalCh)
	}
	for {
		select {}
	}
}
