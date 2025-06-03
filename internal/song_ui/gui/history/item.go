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
	order persistence.Order
}

func NewOrderGui(order persistence.Order) *OrderGui {
	entry, err := persistence.GetEntry(order.ID)
	if err != nil {
		panic(err)
	}
	ig := &OrderGui{
		Entry: entry,
		order: order,
	}

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *OrderGui) CreateRenderer() fyne.WidgetRenderer {
	title := widgets.NewEllipseText(ig.order.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	songInfo := fmt.Sprintf("%s (%s)", ig.order.ID, ig.order.DanceRoom)
	songInfoLine := canvas.NewText(songInfo, theme.Color(theme.ColorNameForeground))
	songInfoLine.TextSize = 12

	orderUser := i18n.T("wrapper_adder", goeasyi18n.Options{
		Data: map[string]any{"Adder": ig.order.Username},
	})
	orderInfo := fmt.Sprintf("%s %s", orderUser, ig.order.Time.Format("15:04:05"))
	orderInfoLine := canvas.NewText(orderInfo, theme.Color(theme.ColorNamePlaceHolder))
	orderInfoLine.TextSize = 12

	thumbnailUrl := ""
	if regexp.MustCompile(`^pypy_`).MatchString(ig.Entry.ID) {
		pypyId, err := strconv.Atoi(ig.Entry.ID[5:])
		if err == nil {
			thumbnailUrl = utils.GetPyPyThumbnailUrl(pypyId)
		}
	}

	return &ItemRenderer{
		ig: ig,

		TitleWidget: title,
		FavoriteBtn: widgets.NewFavoriteBtn(ig.order.ID, ig.order.Title),

		SongInfoLine:  songInfoLine,
		OrderInfoLine: orderInfoLine,

		LocalSong: widgets.NewLocalSongOperations(ig.Entry),
		Thumbnail: widgets.NewThumbnail(thumbnailUrl),
		Separator: widget.NewSeparator(),
	}
}

type ItemRenderer struct {
	ig *OrderGui

	TitleWidget *widgets.EllipseText
	FavoriteBtn *widgets.FavoriteBtn

	SongInfoLine  *canvas.Text
	OrderInfoLine *canvas.Text

	LocalSong *widgets.LocalSongOperations
	Thumbnail *widgets.Thumbnail

	Separator *widget.Separator
}

func (r *ItemRenderer) MinSize() fyne.Size {
	p := theme.Padding()

	minHeight := r.TitleWidget.MinSize().Height + r.SongInfoLine.MinSize().Height
	minHeight += r.OrderInfoLine.MinSize().Height + r.LocalSong.MinSize().Height + p*3
	return fyne.NewSize(300, minHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	favWidth := r.FavoriteBtn.MinSize().Width
	titleHeight := r.TitleWidget.MinSize().Height

	r.FavoriteBtn.Resize(fyne.NewSize(favWidth, favWidth))
	r.FavoriteBtn.Move(fyne.NewPos(size.Width-favWidth-p, (titleHeight-r.FavoriteBtn.MinSize().Height)/2))

	r.TitleWidget.Resize(fyne.NewSize(size.Width-favWidth-p*2, titleHeight))
	r.TitleWidget.Move(fyne.NewPos(p, 0))

	infoHeight := r.LocalSong.MinSize().Height + r.SongInfoLine.MinSize().Height
	infoHeight += r.OrderInfoLine.MinSize().Height + p*2

	imageWidth := float32(0)
	if size.Width > 320 {
		imageWidth = 160
		if size.Width < 400 {
			imageWidth = size.Width - 240
		}
		r.Thumbnail.Resize(fyne.NewSize(imageWidth, infoHeight))
		r.Thumbnail.Move(fyne.NewPos(p, titleHeight))
		r.Thumbnail.Show()

		imageWidth += p
	} else {
		r.Thumbnail.Hide()
	}
	imageWidth += p

	r.SongInfoLine.Resize(r.SongInfoLine.MinSize())
	r.SongInfoLine.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.SongInfoLine.MinSize().Height + p

	r.OrderInfoLine.Resize(r.OrderInfoLine.MinSize())
	r.OrderInfoLine.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.OrderInfoLine.MinSize().Height + p

	r.LocalSong.Resize(r.LocalSong.MinSize())
	r.LocalSong.Move(fyne.NewPos(imageWidth, titleHeight))
	titleHeight += r.LocalSong.MinSize().Height + p

	r.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *ItemRenderer) Refresh() {
	r.TitleWidget.Refresh()
	r.Thumbnail.Refresh()
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Separator,
		r.TitleWidget,
		r.FavoriteBtn,
		r.SongInfoLine,
		r.OrderInfoLine,
		r.LocalSong,
		r.Thumbnail,
	}
}

func (r *ItemRenderer) Destroy() {
}
