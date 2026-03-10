package cache_window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/custom_fyne/containers/lists"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

func configure(list lists.ReusableList[video_cache.LocalVideoInfo], isInPreserved bool, loadedCh chan struct{}) func() {
	subFn := persistence.SubscribeMetaTableChange
	if isInPreserved {
		subFn = persistence.SubscribePreservedListChange
	}

	SubscriberFn := func() *utils.EventSubscriber[lists.ListItemChange] {
		return utils.PipeEvent(subFn(), func(payload persistence.MetaChange) (lists.ListItemChange, bool) {
			if payload.Type == "video" {
				return lists.ListItemChange{Op: payload.Op, ID: payload.ID}, true
			}
			return lists.ListItemChange{}, false
		})
	}
	RendererFn := func(item lists.ListItem[video_cache.LocalVideoInfo]) fyne.WidgetRenderer {
		return newLocalFileRenderer(item, isInPreserved)
	}
	GetDataFn := cache.GetVideoCache().GetLocalVideoInfo
	ListDataFn := func(offset int) []video_cache.LocalVideoInfo {
		return cache.GetVideoCache().ListLocalVideos(offset, "size", isInPreserved)
	}

	list.ConfigureAllStubs(SubscriberFn, RendererFn, GetDataFn, ListDataFn)
	go func() {
		<-loadedCh
		list.RefreshItems()
	}()

	return list.RefreshItems
}

type FileListGui struct {
	widget.BaseWidget

	IsInPreserved bool

	loadedCh chan struct{}
}

func NewFileListGui(isInPreserved bool) *FileListGui {
	g := &FileListGui{
		IsInPreserved: isInPreserved,

		loadedCh: make(chan struct{}, 1),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *FileListGui) CreateRenderer() fyne.WidgetRenderer {
	var list fyne.CanvasObject
	var labelText string

	refreshBtn := button.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	if g.IsInPreserved {
		labelText = i18n.T("label_cache_allow_list")
		l := lists.NewBaseList[video_cache.LocalVideoInfo]()
		refreshBtn.OnClick = configure(l, true, g.loadedCh)
		list = l
	} else {
		labelText = i18n.T("label_cache_local")
		l := lists.NewInfiniteList[video_cache.LocalVideoInfo]()
		refreshBtn.OnClick = configure(l, false, g.loadedCh)
		list = l
	}

	label := canvas.NewText(labelText, theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	r := &fileListRenderer{
		g: g,

		label:      label,
		refreshBtn: refreshBtn,
		list:       list,
	}

	return r
}

type fileListRenderer struct {
	g *FileListGui

	label      *canvas.Text
	refreshBtn fyne.CanvasObject
	list       fyne.CanvasObject
}

func (r *fileListRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (r *fileListRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	btnSize := r.refreshBtn.MinSize().Height
	topHeight := btnSize + p*2

	labelHeight := r.label.MinSize().Height

	r.label.Resize(r.label.MinSize())
	r.label.Move(fyne.NewPos(p*2, (topHeight-labelHeight)/2))
	r.refreshBtn.Resize(r.refreshBtn.MinSize())
	r.refreshBtn.Move(fyne.NewPos(size.Width-btnSize-p, p))

	r.list.Resize(fyne.NewSize(size.Width, size.Height-topHeight))
	r.list.Move(fyne.NewPos(0, topHeight))
}

func (r *fileListRenderer) Refresh() {
	canvas.Refresh(r.g)
}

func (r *fileListRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.list,
		r.label,
		r.refreshBtn,
	}
}

func (r *fileListRenderer) Destroy() {
}
