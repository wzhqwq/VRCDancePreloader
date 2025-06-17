package favorite

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"image/color"
	"regexp"
	"strconv"
)

type ItemGui struct {
	widget.BaseWidget

	Entry *persistence.LocalSongEntry

	entryChanged bool
}

func NewItemGui(entry *persistence.LocalSongEntry) *ItemGui {
	ig := &ItemGui{
		Entry:        entry,
		entryChanged: true,
	}

	ig.ExtendBaseWidget(ig)

	return ig
}

func (ig *ItemGui) UpdateFavoriteEntry(entry *persistence.LocalSongEntry) {
	ig.Entry = entry
	ig.entryChanged = true

	fyne.Do(func() {
		ig.Refresh()
	})
}

func (ig *ItemGui) CreateRenderer() fyne.WidgetRenderer {
	title := widgets.NewEllipseText(ig.Entry.Title, theme.Color(theme.ColorNameForeground))
	title.TextSize = 16
	title.TextStyle = fyne.TextStyle{Bold: true}

	id := canvas.NewText(ig.Entry.ID, color.Gray{128})
	id.Alignment = fyne.TextAlignTrailing
	id.TextSize = 12

	favoriteBtn := button.NewFavoriteBtn(ig.Entry.ID, ig.Entry.Title)
	favoriteBtn.SetMinSquareSize(36)
	favoriteBtn.SetPadding(8)

	actions := button.NewSideActions()
	actions.Buttons = []fyne.CanvasObject{
		favoriteBtn,
	}

	return &ItemRenderer{
		TitleWidget: title,
		IDWidget:    id,
		LocalSong:   widgets.NewLocalSongOperations(ig.Entry),
		Thumbnail:   widgets.NewThumbnail(""),
		Separator:   widget.NewSeparator(),
		Actions:     actions,

		ig: ig,
	}
}

type ItemRenderer struct {
	ig *ItemGui

	TitleWidget *widgets.EllipseText
	IDWidget    *canvas.Text
	LocalSong   *widgets.LocalSongOperations
	Thumbnail   *widgets.Thumbnail
	Separator   *widget.Separator
	Actions     *button.SideActions
}

func (r *ItemRenderer) MinSize() fyne.Size {
	minHeight := r.TitleWidget.MinSize().Height + r.IDWidget.MinSize().Height + r.LocalSong.MinSize().Height + theme.Padding()*2
	return fyne.NewSize(300, minHeight)
}

func (r *ItemRenderer) Layout(size fyne.Size) {
	p := theme.Padding()

	titleHeight := r.TitleWidget.MinSize().Height

	r.TitleWidget.Resize(fyne.NewSize(size.Width, titleHeight))
	r.TitleWidget.Move(fyne.NewPos(0, 0))

	infoHeight := r.LocalSong.MinSize().Height + r.IDWidget.MinSize().Height + p*2

	imageWidth := float32(0)
	if size.Width > 320 {
		imageWidth = 160
		if size.Width < 400 {
			imageWidth = size.Width - 240
		}
		r.Thumbnail.Resize(fyne.NewSize(imageWidth, infoHeight-p*2))
		r.Thumbnail.Move(fyne.NewPos(0, titleHeight+p))
		r.Thumbnail.Show()

		imageWidth += p
	} else {
		r.Thumbnail.Hide()
	}

	infoX := imageWidth
	infoY := titleHeight

	r.IDWidget.Resize(r.IDWidget.MinSize())
	r.IDWidget.Move(fyne.NewPos(infoX, infoY))
	infoY += r.IDWidget.MinSize().Height

	r.LocalSong.Resize(r.LocalSong.MinSize())
	r.LocalSong.Move(fyne.NewPos(infoX, infoY+p))

	r.Separator.Resize(fyne.NewSize(size.Width+p, 1))
	r.Separator.Move(fyne.NewPos(0, size.Height-1))

	r.Actions.Resize(size)
	r.Actions.Move(fyne.NewPos(0, 0))
}

func (r *ItemRenderer) Refresh() {
	if r.ig.entryChanged {
		r.ig.entryChanged = false

		r.TitleWidget.Text = r.ig.Entry.Title
		r.IDWidget.Text = r.ig.Entry.ID

		thumbnailUrl := ""
		if regexp.MustCompile(`^pypy_`).MatchString(r.ig.Entry.ID) {
			pypyId, err := strconv.Atoi(r.ig.Entry.ID[5:])
			if err == nil {
				thumbnailUrl = utils.GetPyPyThumbnailUrl(pypyId)
			}
		}
		r.Thumbnail.LoadImageFromURL(thumbnailUrl)

		r.LocalSong.UpdateEntry(r.ig.Entry)
	}
	r.TitleWidget.Refresh()
	r.Thumbnail.Refresh()

	canvas.Refresh(r.ig)
}

func (r *ItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.TitleWidget,
		r.IDWidget,
		r.LocalSong,
		r.Thumbnail,
		r.Actions,
		r.Separator,
	}
}

func (r *ItemRenderer) Destroy() {
}
