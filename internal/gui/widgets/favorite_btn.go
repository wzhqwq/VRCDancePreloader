package widgets

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type FavoriteBtn struct {
	PaddedIconBtn

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
	b.Extend(nil)

	b.OnClick = func() {
		b.SetFavorite(!b.isFavorite)
		if b.isFavorite {
			persistence.GetFavorite().SetFavorite(b.ID, b.Title)
		} else {
			persistence.GetFavorite().UnsetFavorite(b.ID)
		}
	}

	b.ExtendBaseWidget(b)

	b.SetFavorite(persistence.IsFavorite(id))
	go func() {
		for {
			select {
			case <-b.eventCh:
				b.SetFavorite(persistence.IsFavorite(b.ID))
			case <-b.closeCh:
				return
			}
		}
	}()

	return b
}

func (b *FavoriteBtn) Destroy() {
	close(b.eventCh)
	close(b.closeCh)
}

func (b *FavoriteBtn) SetFavorite(f bool) {
	b.isFavorite = f
	if f {
		b.SetIcon(icons.GetIcon("collection-fill"))
	} else {
		b.SetIcon(icons.GetIcon("collection-grey"))
	}
	b.Refresh()
}
