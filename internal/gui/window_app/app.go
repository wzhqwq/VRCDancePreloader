package window_app

import (
	"os"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

var a fyne.App

func InitFyne() {
	a = app.New()
	a.Settings().SetTheme(&cTheme{})
}

func MainLoop(closeCh chan os.Signal) {
	a.Run()
	closeCh <- syscall.SIGINT
}

func NewWindow(title string) fyne.Window {
	return a.NewWindow(title)
}

func Driver() fyne.Driver {
	return a.Driver()
}
