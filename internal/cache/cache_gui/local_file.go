package cache_gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"log"
)

type LocalFileGui struct {
	widget.BaseWidget

	IsInAllowList bool
	Info          types.CacheFileInfo
}

func NewLocalFileGui(info types.CacheFileInfo, isInAllowList bool) *LocalFileGui {
	g := &LocalFileGui{
		Info:          info,
		IsInAllowList: isInAllowList,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalFileGui) UpdateInfo(info types.CacheFileInfo) {
	g.Info = info
	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *LocalFileGui) getTitle() string {
	title := g.Info.ID + ".mp4"
	if pypyId, ok := utils.CheckIdIsPyPy(g.Info.ID); ok {
		if song, ok := raw_song.FindPyPySong(pypyId); ok {
			title = song.Name
		}
	}
	return title
}

func (g *LocalFileGui) CreateRenderer() fyne.WidgetRenderer {
	titleWidget := widgets.NewEllipseText(g.getTitle(), theme.Color(theme.ColorNameForeground))
	titleWidget.TextSize = 16

	r := &LocalFileGuiRenderer{
		g: g,

		Title:     titleWidget,
		Infos:     container.NewHBox(),
		Buttons:   container.NewHBox(),
		Separator: widget.NewSeparator(),
	}
	r.RefreshButtons()
	r.RefreshInfos()

	return r
}

type LocalFileGuiRenderer struct {
	g *LocalFileGui

	Title     *widgets.EllipseText
	Infos     *fyne.Container
	Separator *widget.Separator

	Buttons *fyne.Container
}

func (r *LocalFileGuiRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	minHeight1 := r.Title.MinSize().Height + r.Infos.MinSize().Height + p
	minHeight := minHeight1 + p*2
	return fyne.NewSize(400, minHeight)
}

func (r *LocalFileGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	titleHeight := r.Title.MinSize().Height
	leftWidth := size.Width - p*5 - r.Buttons.MinSize().Width
	r.Title.Resize(fyne.NewSize(leftWidth, titleHeight))
	r.Title.Move(fyne.NewPos(p, p))

	bottomHeight := size.Height - titleHeight - p*2

	r.Infos.Resize(fyne.NewSize(leftWidth, bottomHeight))
	r.Infos.Move(fyne.NewPos(p, titleHeight+p*2))

	buttonsHeight := r.Buttons.MinSize().Height
	r.Buttons.Resize(r.Buttons.MinSize())
	r.Buttons.Move(fyne.NewPos(leftWidth+p, (size.Height-buttonsHeight)/2))

	r.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *LocalFileGuiRenderer) RefreshInfos() {
	sizeWidget := canvas.NewText(utils.PrettyByteSize(r.g.Info.Size), theme.Color(theme.ColorNamePlaceHolder))
	sizeWidget.TextSize = 12
	r.Infos.Add(sizeWidget)

	if persistence.IsFavorite(r.g.Info.ID) {
		favoriteLabel := canvas.NewText(i18n.T("label_cache_is_favorite"), theme.Color(theme.ColorNamePrimary))
		favoriteLabel.TextSize = 12
		r.Infos.Add(favoriteLabel)
	}
	if !r.g.IsInAllowList && persistence.IsInAllowList(r.g.Info.ID) {
		allowedLabel := canvas.NewText(i18n.T("label_cache_is_preserved"), theme.Color(theme.ColorNamePrimary))
		allowedLabel.TextSize = 12
		r.Infos.Add(allowedLabel)
	}
	if r.g.Info.IsActive {
		activeLabel := canvas.NewText(i18n.T("label_cache_in_use"), theme.Color(theme.ColorNameError))
		activeLabel.TextSize = 12
		r.Infos.Add(activeLabel)
	}
	if r.g.Info.IsPartial {
		partialLabel := canvas.NewText(i18n.T("label_cache_is_partial"), theme.Color(theme.ColorNameWarning))
		partialLabel.TextSize = 12
		r.Infos.Add(partialLabel)
	}
}

func (r *LocalFileGuiRenderer) RefreshButtons() {
	if r.g.IsInAllowList {
		removeFromListBtn := button.NewPaddedIconBtn(theme.WindowCloseIcon())
		removeFromListBtn.SetMinSquareSize(30)
		removeFromListBtn.OnClick = func() {
			persistence.RemoveFromAllowList(r.g.Info.ID)
		}
		r.Buttons.Add(removeFromListBtn)
	} else {
		if !r.g.Info.IsActive {
			deleteBtn := button.NewPaddedIconBtn(theme.DeleteIcon())
			deleteBtn.SetMinSquareSize(30)
			deleteBtn.OnClick = func() {
				err := cache.RemoveLocalCacheById(r.g.Info.ID)
				if err != nil {
					log.Println(err)
				}
			}
			r.Buttons.Add(deleteBtn)
		}

		if !persistence.IsInAllowList(r.g.Info.ID) {
			addAllowListBtn := button.NewPaddedIconBtn(theme.NavigateNextIcon())
			addAllowListBtn.SetMinSquareSize(30)
			addAllowListBtn.OnClick = func() {
				persistence.AddToAllowList(r.g.Info.ID, r.g.Info.Size)
			}
			r.Buttons.Add(addAllowListBtn)
		}
	}
}

func (r *LocalFileGuiRenderer) Refresh() {
	r.Buttons.RemoveAll()
	r.RefreshButtons()
	r.Buttons.Refresh()

	r.Infos.RemoveAll()
	r.RefreshInfos()
	r.Infos.Refresh()

	canvas.Refresh(r.g)
}

func (r *LocalFileGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Title,
		r.Infos,
		r.Buttons,
		r.Separator,
	}
}

func (r *LocalFileGuiRenderer) Destroy() {

}
