package cache_window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

var openedWindow fyne.Window

func OpenCacheWindow() {
	if openedWindow != nil {
		return
	}

	openedWindow = custom_fyne.NewWindow(i18n.T("label_cache_local"))
	localFiles := NewLocalFilesGui()
	allowList := NewAllowListGui()

	localFiles.RefreshFiles()
	allowList.RefreshFiles()

	splitContainer := container.NewGridWithColumns(2, localFiles, allowList)

	openedWindow.SetContent(splitContainer)
	openedWindow.Show()
	openedWindow.SetOnClosed(func() {
		openedWindow = nil
	})
}
