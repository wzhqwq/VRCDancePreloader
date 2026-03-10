package cache_window

import (
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/custom_fyne/containers/lists"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func getTitle(info video_cache.LocalVideoInfo) string {
	id := info.ID()
	entry, err := persistence.GetEntry(id)
	if err == nil {
		return entry.Title
	}

	if pypyId, ok := utils.CheckIdIsPyPy(id); ok {
		if song, ok := raw_song.FindPyPySong(pypyId); ok {
			return song.Name
		}
	}
	if wannaId, ok := utils.CheckIdIsWanna(id); ok {
		if song, ok := raw_song.FindWannaSong(wannaId); ok {
			return song.FullTitle()
		}
	}
	if duduId, ok := utils.CheckIdIsDuDu(id); ok {
		if song, ok := raw_song.FindDuDuSong(duduId); ok {
			return song.FullTitle()
		}
	}
	return id + ".mp4"
}

func newLocalFileRenderer(proxy lists.ListItem[video_cache.LocalVideoInfo], inPreserved bool) fyne.WidgetRenderer {
	info := proxy.Data()

	titleWidget := widgets.NewSongTitle(info.ID(), getTitle(info), theme.Color(theme.ColorNameForeground))
	titleWidget.TextSize = 16

	r := &localFileRenderer{
		proxy: proxy,

		IsInPreserved: inPreserved,

		Title:     titleWidget,
		Infos:     container.NewHBox(),
		Buttons:   container.NewHBox(),
		Separator: widget.NewSeparator(),
	}
	r.RefreshButtons(info)
	r.RefreshInfos(info)

	return r
}

type localFileRenderer struct {
	proxy lists.ListItem[video_cache.LocalVideoInfo]

	IsInPreserved bool

	Title     *widgets.SongTitle
	Infos     *fyne.Container
	Separator *widget.Separator

	Buttons *fyne.Container
}

func (r *localFileRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	minHeight1 := r.Title.MinSize().Height + r.Infos.MinSize().Height + p
	minHeight := minHeight1 + p*2
	return fyne.NewSize(400, minHeight)
}

func (r *localFileRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	titleHeight := r.Title.MinSize().Height
	leftWidth := size.Width - p*6 - r.Buttons.MinSize().Width
	r.Title.Resize(fyne.NewSize(leftWidth, titleHeight))
	r.Title.Move(fyne.NewPos(p*2, p))

	bottomHeight := size.Height - titleHeight - p*2

	r.Infos.Resize(fyne.NewSize(leftWidth, bottomHeight))
	r.Infos.Move(fyne.NewPos(p*3, titleHeight+p*2))

	buttonsHeight := r.Buttons.MinSize().Height
	r.Buttons.Resize(r.Buttons.MinSize())
	r.Buttons.Move(fyne.NewPos(leftWidth+p*3, (size.Height-buttonsHeight)/2))

	r.Separator.Resize(fyne.NewSize(size.Width, 1))
	r.Separator.Move(fyne.NewPos(0, size.Height-1))
}

func (r *localFileRenderer) RefreshInfos(info video_cache.LocalVideoInfo) {
	sizeWidget := canvas.NewText(utils.PrettyByteSize(info.Meta.Size), theme.Color(theme.ColorNamePlaceHolder))
	sizeWidget.TextSize = 12
	r.Infos.Add(sizeWidget)

	if persistence.IsFavorite(info.ID()) {
		favoriteLabel := canvas.NewText(i18n.T("label_cache_is_favorite"), theme.Color(theme.ColorNamePrimary))
		favoriteLabel.TextSize = 12
		r.Infos.Add(favoriteLabel)
	}
	if !r.IsInPreserved && info.Meta.Preserved {
		allowedLabel := canvas.NewText(i18n.T("label_cache_is_preserved"), theme.Color(theme.ColorNamePrimary))
		allowedLabel.TextSize = 12
		r.Infos.Add(allowedLabel)
	}
	if info.Active {
		activeLabel := canvas.NewText(i18n.T("label_cache_in_use"), theme.Color(theme.ColorNameError))
		activeLabel.TextSize = 12
		r.Infos.Add(activeLabel)
	}
	if info.Meta.Partial {
		partialLabel := canvas.NewText(i18n.T("label_cache_is_partial"), theme.Color(theme.ColorNameWarning))
		partialLabel.TextSize = 12
		r.Infos.Add(partialLabel)
	}
}

func (r *localFileRenderer) PrintRemoved() {
	r.Title.Color = theme.Color(theme.ColorNamePlaceHolder)
	r.Title.TextStyle.Italic = true
	r.Title.Refresh()

	tipWidget := canvas.NewText(i18n.T("label_cache_removed"), theme.Color(theme.ColorNamePlaceHolder))
	tipWidget.TextSize = 12
	r.Infos.Add(tipWidget)
}

func (r *localFileRenderer) RefreshButtons(info video_cache.LocalVideoInfo) {
	if r.IsInPreserved {
		removeFromListBtn := button.NewPaddedIconBtn(theme.WindowCloseIcon())
		removeFromListBtn.SetMinSquareSize(30)
		removeFromListBtn.OnClick = func() {
			info.Meta.SetPreserved(false)
		}
		r.Buttons.Add(removeFromListBtn)
	} else {
		if !info.Active {
			deleteBtn := button.NewPaddedIconBtn(theme.DeleteIcon())
			deleteBtn.SetMinSquareSize(30)
			deleteBtn.OnClick = func() {
				err := cache.RemoveLocalCacheById(info.ID())
				if err != nil {
					log.Println(err)
				}
			}
			r.Buttons.Add(deleteBtn)
		}

		if !info.Meta.Preserved {
			addAllowListBtn := button.NewPaddedIconBtn(theme.NavigateNextIcon())
			addAllowListBtn.SetMinSquareSize(30)
			addAllowListBtn.OnClick = func() {
				info.Meta.SetPreserved(true)
			}
			r.Buttons.Add(addAllowListBtn)
		}
	}
}

func (r *localFileRenderer) Refresh() {
	info := r.proxy.Data()

	if r.proxy.Removed() {
		r.Buttons.RemoveAll()
		r.Buttons.Refresh()

		r.Infos.RemoveAll()
		r.PrintRemoved()
		r.Infos.Refresh()
	}

	if r.proxy.Dirty() {
		r.Buttons.RemoveAll()
		r.RefreshButtons(info)
		r.Buttons.Refresh()

		r.Infos.RemoveAll()
		r.RefreshInfos(info)
		r.Infos.Refresh()
	}
}

func (r *localFileRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Title,
		r.Infos,
		r.Buttons,
		r.Separator,
	}
}

func (r *localFileRenderer) Destroy() {

}
