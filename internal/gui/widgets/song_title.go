package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"image/color"
	"log"
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

		persistence.GetLocalSongs().AddLocalSongIfNotExist(id, completedTitle)
		entry, err := persistence.GetEntry(id)
		if err != nil {
			log.Println("failed to get entry: " + err.Error())
			return
		}

		if entry.Title != completedTitle {
			entry.UpdateTitle(completedTitle)
		}
		t.Text = entry.Title

		fyne.Do(func() {
			t.Refresh()
		})
	}()

	return t
}
