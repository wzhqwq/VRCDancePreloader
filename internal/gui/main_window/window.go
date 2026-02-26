package main_window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/global_state"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/favorite"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/history"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/main_window/settings"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

func Start() {
	custom_fyne.InitFyne()
	MainWindow()

	if path := global_state.GetDbMigrationPath(); path != "" {
		dialog.NewInformation(
			i18n.T("message_title_db_migrated"),
			i18n.T("message_db_migration", goeasyi18n.Options{
				Data: map[string]interface{}{
					"Dir": path,
				},
			}),
			custom_fyne.GetParent(),
		).Show()
	}
}
func Stop() {
}

func MainWindow() fyne.Window {
	w := custom_fyne.NewMainWindow(i18n.T("app_name"))

	playlistGui := playlist.NewPlaylistManager()
	historyGui := history.NewGui()
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
