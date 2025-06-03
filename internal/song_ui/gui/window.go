package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/favorite"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/history"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/playlist"
)

func Start() {
	window_app.InitFyne()
	MainWindow()
}
func Stop() {
}

func MainWindow() fyne.Window {
	w := window_app.NewWindow(i18n.T("app_name"))
	w.SetMaster()

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("btn_playlist"), playlist.NewPlaylistManager()),
		container.NewTabItem(i18n.T("btn_history"), history.NewHistoryGui()),
		container.NewTabItem(i18n.T("btn_favorites"), favorite.NewFavoritesGui()),
		container.NewTabItem(i18n.T("btn_settings"), config.CreateSettingsContainer()),
	)
	w.SetContent(tabs)
	w.SetPadded(false)

	w.Resize(fyne.NewSize(350, 500))

	w.Show()

	history.CheckRecordContinuity(w)

	return w
}
