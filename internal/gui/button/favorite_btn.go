package button

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

type FavoriteBtn struct {
	PaddedIconBtn

	ID    string
	Title string

	isFavorite bool

	destroyed bool
}

func NewFavoriteBtn(id, title string) *FavoriteBtn {
	b := &FavoriteBtn{
		ID:    id,
		Title: title,
	}
	b.Extend(nil)

	b.OnClick = func() {
		b.SetFavorite(!b.isFavorite)
		if b.isFavorite {
			persistence.GetLocalSongs().SetFavorite(b.ID, b.Title)
		} else {
			persistence.GetLocalSongs().UnsetFavorite(b.ID)
		}
	}
	b.AddLifeCycleFn(b.loop)

	b.ExtendBaseWidget(b)

	b.SetFavorite(persistence.IsFavorite(id))

	return b
}

func (b *FavoriteBtn) loop(stopCh <-chan struct{}) {
	ch := persistence.GetLocalSongs().SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case <-ch.Channel:
			b.SetFavorite(persistence.IsFavorite(b.ID))
		case <-stopCh:
			return
		}
	}
}

func (b *FavoriteBtn) SetFavorite(f bool) {
	b.isFavorite = f
	if f {
		b.SetIcon(icons.GetIcon("star-fill"))
	} else {
		b.SetIcon(icons.GetIcon("star-grey"))
	}
}
