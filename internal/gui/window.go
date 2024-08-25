package gui

import (
	"os"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
)

var a fyne.App

func InitGui() {
	a = app.New()
	a.Settings().SetTheme(&cTheme{})
	MainWindow()
}

func MainWindow() {
	w := a.NewWindow("Hello")

	pl := NewPlayListGui()
	go pl.drawFromChannels()

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("btn_playlist"), pl.Container),
		container.NewTabItem(i18n.T("btn_history"), widget.NewLabel("Not Implemented")),
	)
	w.SetContent(tabs)
	w.SetPadded(false)

	w.Show()
}

func MainLoop(closeCh chan os.Signal) {
	a.Run()
	closeCh <- syscall.SIGINT
}
