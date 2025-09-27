package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

type SongTitle struct {
	EllipseText

	ID string
}

func NewSongTitle(id, title string, color color.Color) *SongTitle {
	t := &SongTitle{
		EllipseText: EllipseText{
			Text:     title,
			Color:    color,
			TextSize: theme.TextSize(),
		},
		ID: id,
	}
	t.ExtendBaseWidget(t)

	go func() {
		completedTitle := third_party_api.CompleteTitleByInternalID(id, title).Get()
		if completedTitle == "" || completedTitle == title {
			return
		}

		persistence.UpdateSavedTitle(id, completedTitle)
		t.Text = completedTitle

		fyne.Do(func() {
			t.Refresh()
		})
	}()

	return t
}
