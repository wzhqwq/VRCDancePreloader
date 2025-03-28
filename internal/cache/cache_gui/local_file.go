package cache_gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
)

type LocalFileGui struct {
	widget.BaseWidget

	Title     *widgets.EllipseText
	Infos     fyne.CanvasObject
	Separator *widget.Separator

	Buttons fyne.CanvasObject
}

func NewLocalFileGui(id string, size int64, isActive bool) *LocalFileGui {
	title := id + ".mp4"
	if pypyId, ok := utils.CheckIdIsPyPy(id); ok {
		if song, ok := raw_song.FindPyPySong(pypyId); ok {
			title = song.Name
		}
	}

	titleWidget := widgets.NewEllipseText(title, theme.Color(theme.ColorNameForeground))
	titleWidget.TextSize = 16

	sizeWidget := canvas.NewText(utils.PrettyByteSize(size), theme.Color(theme.ColorNamePlaceHolder))
	sizeWidget.TextSize = 12

	infos := container.NewHBox(
		sizeWidget,
	)

	if persistence.GetFavorite().IsFavorite(id) {
		favoriteLabel := canvas.NewText(i18n.T("label_cache_is_favorite"), theme.Color(theme.ColorNamePrimary))
		favoriteLabel.TextSize = 12
		infos.Add(favoriteLabel)
	}

	buttons := container.NewHBox()

	if isActive {
		activeLabel := canvas.NewText(i18n.T("label_cache_in_use"), theme.Color(theme.ColorNameError))
		activeLabel.TextSize = 12
		infos.Add(activeLabel)
	} else {
		deleteBtn := widgets.NewPaddedIconBtn(theme.DeleteIcon())
		deleteBtn.SetMinSquareSize(30)
		deleteBtn.OnClick = func() {
			err := cache.RemoveLocalCacheById(id)
			if err != nil {
				log.Println(err)
			}
		}
		buttons.Add(deleteBtn)
	}

	addAllowListBtn := widgets.NewPaddedIconBtn(theme.NavigateNextIcon())
	addAllowListBtn.SetMinSquareSize(30)
	addAllowListBtn.OnClick = func() {
		// TODO add to allow list
	}
	buttons.Add(addAllowListBtn)

	g := &LocalFileGui{
		Title:     titleWidget,
		Infos:     infos,
		Buttons:   buttons,
		Separator: widget.NewSeparator(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalFileGui) CreateRenderer() fyne.WidgetRenderer {
	return &LocalFileGuiRenderer{
		g: g,
	}
}

type LocalFileGuiRenderer struct {
	g *LocalFileGui
}

func (r *LocalFileGuiRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	minHeight1 := r.g.Title.MinSize().Height + r.g.Infos.MinSize().Height + p
	minHeight := minHeight1 + p*2
	return fyne.NewSize(400, minHeight)
}

func (r *LocalFileGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	titleHeight := r.g.Title.MinSize().Height
	leftWidth := size.Width - p*5 - r.g.Buttons.MinSize().Width
	r.g.Title.Resize(fyne.NewSize(leftWidth, titleHeight))
	r.g.Title.Move(fyne.NewPos(p, p))

	bottomHeight := size.Height - titleHeight - p*2

	r.g.Infos.Resize(fyne.NewSize(leftWidth, bottomHeight))
	r.g.Infos.Move(fyne.NewPos(p, titleHeight+p*2))

	buttonsHeight := r.g.Buttons.MinSize().Height
	r.g.Buttons.Resize(r.g.Buttons.MinSize())
	r.g.Buttons.Move(fyne.NewPos(leftWidth+p, (size.Height-buttonsHeight)/2))

	r.g.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.g.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *LocalFileGuiRenderer) Refresh() {
	r.g.Title.Refresh()
}

func (r *LocalFileGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.Title,
		r.g.Infos,
		r.g.Buttons,
		r.g.Separator,
	}
}

func (r *LocalFileGuiRenderer) Destroy() {

}
