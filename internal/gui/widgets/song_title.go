package widgets

import (
	"image/color"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
)

var placeholderTitleRegex = regexp.MustCompile(`^(?:(?:PyPy|Wanna|DuDuFit)Dance|YouTube|BiliBili)\s`)

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
	t.AddLifeCycleFn(t.loop)

	return t
}

func (t *SongTitle) loop(stopCh <-chan struct{}) {
	if !placeholderTitleRegex.MatchString(t.Text) && !strings.HasSuffix(t.Text, ".mp4") {
		return
	}
	provider := third_parties.GetProviderById(t.ID)
	if provider == nil {
		return
	}

	infoHandle := provider.Info(t.ID)
	defer infoHandle.Release()

	ch := infoHandle.Subscribe()
	defer ch.Close()

	title := infoHandle.Snapshot().Data.Title
	persistence.UpdateSavedTitle(t.ID, title)
	t.Text = title
	fyne.Do(t.Refresh)

	for {
		select {
		case <-stopCh:
			return
		case <-ch.Channel:
			// The notification is only a wake-up: read the entry instead of the
			// payload, so what is rendered is the current title (remote.go's
			// deliveries are ordered, but a newer state may exist by now).
			snap := infoHandle.Snapshot()
			if snap.Status.Valid() {
				title = snap.Data.Title
				persistence.UpdateSavedTitle(t.ID, title)
				t.Text = title
				fyne.Do(t.Refresh)
			}
		}
	}
}
