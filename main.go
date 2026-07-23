package main

import (
	"runtime/debug"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
	"github.com/wzhqwq/VRCDancePreloader/internal/tui"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"

	"os"
	"os/signal"
	"syscall"

	"github.com/alexflint/go-arg"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var buildGuiOn = false

var logger = utils.NewLogger("VRCDP")

var args struct {
	GuiEnabled bool `arg:"-g,--gui" default:"false" help:"enable GUI"`
	TuiEnabled bool `arg:"-t,--tui" default:"false" help:"enable TUI"`
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			logger.ErrorLn("panicked:", err)
			logger.DebugLn(string(debug.Stack()))
		}
	}()

	arg.MustParse(&args)

	custom_fyne.InitRoot()

	// Apply build tag
	if buildGuiOn {
		args.GuiEnabled = true
	}

	i18n.Init()

	// Listen for interrupt
	osSignalCh := make(chan os.Signal, 1)
	signal.Notify(osSignalCh, syscall.SIGINT, syscall.SIGTERM)

	migrated := host.StartHost()
	defer host.Shutdown()

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
		main_window.Start(migrated)
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
