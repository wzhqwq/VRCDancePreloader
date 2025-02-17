package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type FavoriteBtn struct {
	widget.Icon
	fyne.Tappable

	ID    string
	Title string

	isFavorite bool

	eventCh chan string
	closeCh chan struct{}
}

func NewFavoriteBtn(id, title string) *FavoriteBtn {
	b := &FavoriteBtn{
		ID:    id,
		Title: title,

		eventCh: persistence.GetFavorite().SubscribeEvent(),
	}
	b.ExtendBaseWidget(b)

	b.SetFavorite(persistence.GetFavorite().IsFavorite(id))
	go func() {
		for {
			select {
			case <-b.eventCh:
				b.SetFavorite(persistence.GetFavorite().IsFavorite(b.ID))
			case <-b.closeCh:
				return
			}
		}
	}()

	return b
}

func (b *FavoriteBtn) Tapped(_ *fyne.PointEvent) {
	b.SetFavorite(!b.isFavorite)
	if b.isFavorite {
		persistence.GetFavorite().SetFavorite(b.ID, b.Title)
	} else {
		persistence.GetFavorite().UnsetFavorite(b.ID)
	}
}

func (b *FavoriteBtn) Destroy() {
	close(b.eventCh)
	close(b.closeCh)
}

func (b *FavoriteBtn) SetFavorite(f bool) {
	b.isFavorite = f
	if f {
		b.Resource = icons.GetIcon("collection-fill")
	} else {
		b.Resource = icons.GetIcon("collection-grey")
	}
	b.Refresh()
}
