package cache_gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var openedWindow fyne.Window

func OpenCacheWindow() {
	if openedWindow != nil {
		return
	}

	openedWindow = window_app.NewWindow(i18n.T("label_cache_local"))
	localFiles := NewLocalFilesGui()
	allowList := NewAllowListGui()

	splitContainer := container.NewGridWithColumns(2, localFiles, allowList)

	openedWindow.SetContent(splitContainer)
	openedWindow.Show()
	openedWindow.SetOnClosed(func() {
		openedWindow = nil
	})
}
