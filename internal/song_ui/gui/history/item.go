package history

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"regexp"
	"strconv"
)

type OrderGui struct {
	widget.BaseWidget

	Entry *persistence.LocalSongEntry

	TitleWidget *widgets.EllipseText
	FavoriteBtn *widgets.FavoriteBtn

	SongInfoLine  *canvas.Text
	OrderInfoLine *canvas.Text

	LocalSong *widgets.LocalSongOperations
	Thumbnail *widgets.Thumbnail

	Separator *widget.Separator
}

func NewOrderGui(order persistence.Order) *OrderGui {
	entry, err := persistence.GetEntry(order.ID)
	if err != nil {
		panic(err)
	}
	title := widgets.NewEllipseText(order.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	songInfo := fmt.Sprintf("%s (%s)", order.ID, order.DanceRoom)
	songInfoLine := canvas.NewText(songInfo, theme.Color(theme.ColorNameForeground))
	songInfoLine.TextSize = 12

	orderUser := i18n.T("wrapper_adder", goeasyi18n.Options{
		Data: map[string]any{"Adder": order.Username},
	})
	orderInfo := fmt.Sprintf("%s %s", orderUser, order.Time.Format("15:04:05"))
	orderInfoLine := canvas.NewText(orderInfo, theme.Color(theme.ColorNamePlaceHolder))
	orderInfoLine.TextSize = 12

	thumbnailUrl := ""
	if regexp.MustCompile(`^pypy_`).MatchString(entry.ID) {
		pypyId, err := strconv.Atoi(entry.ID[5:])
		if err == nil {
			thumbnailUrl = utils.GetPyPyThumbnailUrl(pypyId)
		}
	}

	ig := &OrderGui{
		Entry: entry,

		TitleWidget: title,
		FavoriteBtn: widgets.NewFavoriteBtn(order.ID, order.Title),

		SongInfoLine:  songInfoLine,
		OrderInfoLine: orderInfoLine,

		LocalSong: widgets.NewLocalSongOperations(entry),
		Thumbnail: widgets.NewThumbnail(thumbnailUrl),
		Separator: widget.NewSeparator(),
	}

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *OrderGui) CreateRenderer() fyne.WidgetRenderer {
	return &ItemRenderer{
		ig: ig,
	}
}

type ItemRenderer struct {
	ig *OrderGui
}

func (r *ItemRenderer) MinSize() fyne.Size {
	p := theme.Padding()

	minHeight := r.ig.TitleWidget.MinSize().Height + r.ig.SongInfoLine.MinSize().Height
	minHeight += r.ig.OrderInfoLine.MinSize().Height + r.ig.LocalSong.MinSize().Height + p*3
	return fyne.NewSize(300, minHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	favWidth := r.ig.FavoriteBtn.MinSize().Width
	titleHeight := r.ig.TitleWidget.MinSize().Height

	r.ig.FavoriteBtn.Resize(fyne.NewSize(favWidth, favWidth))
	r.ig.FavoriteBtn.Move(fyne.NewPos(size.Width-favWidth-p, (titleHeight-r.ig.FavoriteBtn.MinSize().Height)/2))

	r.ig.TitleWidget.Resize(fyne.NewSize(size.Width-favWidth-p*2, titleHeight))
	r.ig.TitleWidget.Move(fyne.NewPos(p, 0))

	infoHeight := r.ig.LocalSong.MinSize().Height + r.ig.SongInfoLine.MinSize().Height
	infoHeight += r.ig.OrderInfoLine.MinSize().Height + p*2

	imageWidth := float32(0)
	if size.Width > 320 {
		imageWidth = 160
		if size.Width < 400 {
			imageWidth = size.Width - 240
		}
		r.ig.Thumbnail.Resize(fyne.NewSize(imageWidth, infoHeight))
		r.ig.Thumbnail.Move(fyne.NewPos(p, titleHeight))
		r.ig.Thumbnail.Show()

		imageWidth += p
	} else {
		r.ig.Thumbnail.Hide()
	}
	imageWidth += p

	r.ig.SongInfoLine.Resize(r.ig.SongInfoLine.MinSize())
	r.ig.SongInfoLine.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.SongInfoLine.MinSize().Height + p

	r.ig.OrderInfoLine.Resize(r.ig.OrderInfoLine.MinSize())
	r.ig.OrderInfoLine.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.OrderInfoLine.MinSize().Height + p

	r.ig.LocalSong.Resize(r.ig.LocalSong.MinSize())
	r.ig.LocalSong.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.ig.LocalSong.MinSize().Height + p

	r.ig.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.ig.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *ItemRenderer) Refresh() {
	r.ig.TitleWidget.Refresh()
	r.ig.Thumbnail.Refresh()
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.ig.Separator,
		r.ig.TitleWidget,
		r.ig.FavoriteBtn,
		r.ig.SongInfoLine,
		r.ig.OrderInfoLine,
		r.ig.LocalSong,
		r.ig.Thumbnail,
	}
}

func (r *ItemRenderer) Destroy() {
}
