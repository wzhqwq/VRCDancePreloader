package playlist

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

type SongListButton struct {
	button.PaddedIconBtn

	labelMap map[string]*widget.Label

	closeCh chan struct{}
}

func getLabelText(room string) string {
	switch room {
	case "PyPyDance":
		if t, ok := raw_song.GetPyPyUpdateTime(); ok {
			return i18n.T("label_song_list_downloaded", goeasyi18n.Options{
				Data: map[string]any{
					"Name": room,
					"Time": t.Format("2006-01-02 15:04:05"),
				}},
			)
		}
	case "WannaDance":
		if t, ok := raw_song.GetWannaUpdateTime(); ok {
			return i18n.T("label_song_list_downloaded", goeasyi18n.Options{
				Data: map[string]any{
					"Name": room,
					"Time": t.Format("2006-01-02 15:04:05"),
				}},
			)
		}
	}
	return i18n.T("label_song_list_downloading", goeasyi18n.Options{
		Data: map[string]any{
			"Name": room,
		}},
	)
}

func isAllSongListComplete() bool {
	if _, ok := raw_song.GetPyPyUpdateTime(); !ok {
		return false
	}
	if _, ok := raw_song.GetWannaUpdateTime(); !ok {
		return false
	}
	return true
}

func NewSongListButton() *SongListButton {
	wholeContent := container.NewVBox()

	scroll := container.NewVScroll(container.NewPadded(wholeContent))
	scroll.SetMinSize(fyne.NewSize(250, 300))

	labelMap := map[string]*widget.Label{
		"PyPyDance":  widget.NewLabel(getLabelText("PyPyDance")),
		"WannaDance": widget.NewLabel(getLabelText("WannaDance")),
	}

	btn := &SongListButton{
		labelMap: labelMap,

		closeCh: make(chan struct{}),
	}
	btn.Extend(nil)

	btn.OnClick = func() {
		openSongListModal(scroll)
	}

	wholeContent.Add(labelMap["PyPyDance"])
	wholeContent.Add(labelMap["WannaDance"])

	btn.OnDestroy = func() {
		close(btn.closeCh)
	}

	go btn.eventLoop()

	btn.ExtendBaseWidget(btn)

	btn.SetComplete(isAllSongListComplete())

	return btn
}

func (b *SongListButton) eventLoop() {
	ch := raw_song.SubscribeSongListChange()
	defer ch.Close()

	for {
		select {
		case <-b.closeCh:
			return
		case name := <-ch.Channel:
			fyne.Do(func() {
				b.labelMap[name].SetText(getLabelText(name))
			})
			b.SetComplete(isAllSongListComplete())
		}
	}
}

func (b *SongListButton) SetComplete(complete bool) {
	if complete {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("song-list"), theme.ColorNameSuccess))
	} else {
		b.SetIcon(theme.NewColoredResource(icons.GetIcon("song-list"), theme.ColorNameWarning))
	}
}

func openSongListModal(content fyne.CanvasObject) {
	dialog.NewCustom(
		i18n.T("message_title_song_list"),
		i18n.T("btn_close"),
		content,
		custom_fyne.GetParent(),
	).Show()
}
