package gui

import (
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
)

func Start() {
	window_app.InitFyne()
	MainWindow()
}

func MainWindow() {
	w := window_app.NewWindow(i18n.T("app_name"))

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
