package history

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

type OrderGui struct {
	widget.BaseWidget

	entry *persistence.LocalSongEntry
	order persistence.Order

	onRemove func()
}

func NewOrderGui(order persistence.Order) *OrderGui {
	entry, err := persistence.GetEntry(order.ID)
	if err != nil {
		// try to recover unsaved information from song list
		persistence.GetLocalSongs().AddLocalSongIfNotExist(order.ID, order.Title)
		entry, err = persistence.GetEntry(order.ID)
		if err != nil {
			// it can't be
			panic(errors.New("failed to add song to local database"))
		}
	}
	ig := &OrderGui{
		entry: entry,
		order: order,
	}

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *OrderGui) CreateRenderer() fyne.WidgetRenderer {
	title := widgets.NewSongTitle(ig.entry.ID, ig.entry.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	songInfo := fmt.Sprintf("%s (%s)", ig.entry.ID, ig.order.DanceRoom)
	songInfoLine := canvas.NewText(songInfo, theme.Color(theme.ColorNameForeground))
	songInfoLine.TextSize = 12

	orderUser := i18n.T("wrapper_adder", goeasyi18n.Options{
		Data: map[string]any{"Adder": ig.order.Username},
	})
	orderInfo := fmt.Sprintf("%s %s", orderUser, ig.order.Time.Format("15:04:05"))
	orderInfoLine := canvas.NewText(orderInfo, theme.Color(theme.ColorNamePlaceHolder))
	orderInfoLine.TextSize = 12

	favoriteBtn := button.NewFavoriteBtn(ig.entry.ID, ig.entry.Title)
	favoriteBtn.SetMinSquareSize(36)
	favoriteBtn.SetPadding(8)

	deleteBtn := button.NewPaddedIconBtn(icons.GetIcon("delete"))
	deleteBtn.SetMinSquareSize(36)
	deleteBtn.SetPadding(8)
	deleteBtn.OnClick = func() {
		if ig.onRemove != nil {
			ig.onRemove()
		}
	}

	actions := button.NewSideActions()
	actions.Buttons = []fyne.CanvasObject{
		favoriteBtn,
		deleteBtn,
	}

	return &ItemRenderer{
		ig: ig,

		TitleWidget: title,

		SongInfoLine:  songInfoLine,
		OrderInfoLine: orderInfoLine,

		LocalSong: widgets.NewLocalSongOperations(ig.entry),
		Thumbnail: widgets.NewThumbnailWithID(ig.entry.ID),
		Separator: widget.NewSeparator(),
		Actions:   actions,
	}
}

type ItemRenderer struct {
	ig *OrderGui

	TitleWidget *widgets.SongTitle

	SongInfoLine  *canvas.Text
	OrderInfoLine *canvas.Text

	LocalSong *widgets.LocalSongOperations
	Thumbnail *widgets.Thumbnail

	Separator *widget.Separator
	Actions   *button.SideActions
}

func (r *ItemRenderer) MinSize() fyne.Size {
	p := theme.Padding()

	minHeight := r.TitleWidget.MinSize().Height + r.SongInfoLine.MinSize().Height
	minHeight += r.OrderInfoLine.MinSize().Height + r.LocalSong.MinSize().Height + p*3
	return fyne.NewSize(300, minHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	titleHeight := r.TitleWidget.MinSize().Height

	r.TitleWidget.Resize(fyne.NewSize(size.Width-p*2, titleHeight))
	r.TitleWidget.Move(fyne.NewPos(p, 0))

	infoHeight := r.LocalSong.MinSize().Height + r.SongInfoLine.MinSize().Height
	infoHeight += r.OrderInfoLine.MinSize().Height + p*2

	imageWidth := float32(0)
	if size.Width > 320 {
		imageWidth = 150
		if size.Width < 400 {
			imageWidth = size.Width - 250
		}
		r.Thumbnail.Resize(fyne.NewSize(imageWidth, infoHeight-p))
		r.Thumbnail.Move(fyne.NewPos(p, titleHeight+p))
		r.Thumbnail.Show()

		imageWidth += p
	} else {
		r.Thumbnail.Hide()
	}

	infoX := imageWidth + p
	infoY := titleHeight

	r.SongInfoLine.Resize(r.SongInfoLine.MinSize())
	r.SongInfoLine.Move(fyne.NewPos(infoX, infoY))
	infoY += r.SongInfoLine.MinSize().Height + p

	r.OrderInfoLine.Resize(r.OrderInfoLine.MinSize())
	r.OrderInfoLine.Move(fyne.NewPos(infoX, infoY))
	infoY += r.OrderInfoLine.MinSize().Height + p

	r.LocalSong.Resize(r.LocalSong.MinSize())
	r.LocalSong.Move(fyne.NewPos(infoX, infoY))
	infoY += r.LocalSong.MinSize().Height + p

	r.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.Separator.Move(fyne.NewPos(0, size.Height-1))

	r.Actions.Resize(size)
	r.Actions.Move(fyne.NewPos(0, 0))
}

func (r *ItemRenderer) Refresh() {
	r.TitleWidget.Refresh()
	r.Thumbnail.Refresh()
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.TitleWidget,
		r.SongInfoLine,
		r.OrderInfoLine,
		r.LocalSong,
		r.Thumbnail,
		r.Actions,
		r.Separator,
	}
}

func (r *ItemRenderer) Destroy() {
}
