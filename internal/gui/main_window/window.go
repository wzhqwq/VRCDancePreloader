package main_window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/favorite"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/history"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/settings"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

func Start() {
	window_app.InitFyne()
	MainWindow()
}
func Stop() {
}

func MainWindow() fyne.Window {
	w := window_app.NewMainWindow(i18n.T("app_name"))

	playlistGui := playlist.NewPlaylistManager()
	historyGui := history.NewHistoryGui()
	favoritesGui := favorite.NewFavoritesGui()
	settingsGui := settings.CreateSettingsContainer()

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("btn_playlist"), playlistGui),
		container.NewTabItem(i18n.T("btn_history"), historyGui),
		container.NewTabItem(i18n.T("btn_favorites"), favoritesGui),
		container.NewTabItem(i18n.T("btn_settings"), settingsGui),
	)
	tabs.OnSelected = func(item *container.TabItem) {
		if tabs.SelectedIndex() == 2 {
			favoritesGui.Activate()
		}
	}
	w.SetContent(tabs)
	w.SetPadded(false)

	w.Resize(fyne.NewSize(350, 500))

	w.Show()

	history.CheckRecordContinuity(w)

	return w
}
