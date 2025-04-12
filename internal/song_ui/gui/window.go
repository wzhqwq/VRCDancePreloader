package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/favorite"
	"github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/history"
	playlistgui "github.com/wzhqwq/VRCDancePreloader/internal/song_ui/gui/playlist"
)

var currentGui *playlistgui.PlayListGui
var stopCh chan struct{}

func Start() {
	window_app.InitFyne()
	playlistContainer := container.NewStack()
	w := MainWindow(playlistContainer)

	ch := playlist.SubscribeNewListEvent()
	stopCh = make(chan struct{})
	go func() {
		defer w.Close()
		for {
			select {
			case <-stopCh:
				if currentGui != nil {
					currentGui.StopCh <- struct{}{}
				}
				return
			case pl := <-ch:
				if currentGui != nil {
					currentGui.StopCh <- struct{}{}
					playlistContainer.Remove(currentGui.Container)
				}
				currentGui = playlistgui.NewPlayListGui(pl)
				playlistContainer.Add(currentGui.Container)
				go currentGui.RenderLoop()
			}
		}
	}()

	// temporarily entry point for record init
	if r := persistence.GetLocalRecords().GetNearestRecord(); r != nil {
		dialog.NewConfirm(
			i18n.T("message_title_continue_record"),
			i18n.T("message_continue_record", goeasyi18n.Options{
				Data: map[string]any{"Time": r.StartTime.Format("15:04:05")},
			}),
			func(confirmed bool) {
				persistence.PrepareHistory(confirmed)
			},
			w,
		).Show()
	} else {
		persistence.PrepareHistory(false)
	}
}
func Stop() {
	stopCh <- struct{}{}
}

func MainWindow(playlistContainer fyne.CanvasObject) fyne.Window {
	w := window_app.NewWindow(i18n.T("app_name"))

	tabs := container.NewAppTabs(
		container.NewTabItem(i18n.T("btn_playlist"), playlistContainer),
		container.NewTabItem(i18n.T("btn_history"), history.NewHistoryGui()),
		container.NewTabItem(i18n.T("btn_favorites"), favorite.NewFavoritesPage()),
		container.NewTabItem(i18n.T("btn_settings"), config.CreateSettingsContainer()),
	)
	w.SetContent(tabs)
	w.SetPadded(false)

	w.Resize(fyne.NewSize(350, 500))

	w.Show()

	return w
}
