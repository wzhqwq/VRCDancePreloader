package button

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type FavoriteBtn struct {
	PaddedIconBtn

	ID    string
	Title string

	isFavorite bool

	favoriteChange *utils.StringEventSubscriber
	closeCh        chan struct{}

	destroyed bool
}

func NewFavoriteBtn(id, title string) *FavoriteBtn {
	b := &FavoriteBtn{
		ID:    id,
		Title: title,

		favoriteChange: persistence.GetLocalSongs().SubscribeEvent(),
		closeCh:        make(chan struct{}),
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
	b.OnDestroy = func() {
		if b.destroyed {
			return
		}
		b.destroyed = true
		close(b.closeCh)
		b.favoriteChange.Close()
	}

	b.ExtendBaseWidget(b)

	b.SetFavorite(persistence.IsFavorite(id))
	go func() {
		for {
			select {
			case <-b.favoriteChange.Channel:
				b.SetFavorite(persistence.IsFavorite(b.ID))
			case <-b.closeCh:
				return
			}
		}
	}()

	return b
}

func (b *FavoriteBtn) SetFavorite(f bool) {
	b.isFavorite = f
	if f {
		b.SetIcon(icons.GetIcon("star-fill"))
	} else {
		b.SetIcon(icons.GetIcon("star-grey"))
	}
}
