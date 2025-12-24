package cache_window

import (
	"weak"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/button"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
)

type LocalFilesGui struct {
	widget.BaseWidget

	infos     []types.CacheFileInfo
	changedId string

	stopCh chan struct{}
}

func NewLocalFilesGui() *LocalFilesGui {
	g := &LocalFilesGui{
		stopCh: make(chan struct{}),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalFilesGui) RenderLoop() {
	localCh := cache.SubscribeLocalFileEvent()
	defer localCh.Close()
	allowCh := persistence.GetAllowList().SubscribeEvent()
	defer allowCh.Close()

	for {
		select {
		case <-g.stopCh:
			return
		case message := <-localCh.Channel:
			if message[0] == '*' {
				g.changedId = message[1:]
				fyne.Do(func() {
					g.Refresh()
				})
			} else {
				g.RefreshFiles()
			}
		case message := <-allowCh.Channel:
			if message[0] == '*' {
				g.changedId = message[1:]
				fyne.Do(func() {
					g.Refresh()
				})
			} else {
				g.RefreshFiles()
			}
		}
	}
}

func (g *LocalFilesGui) CreateRenderer() fyne.WidgetRenderer {
	list := container.NewVBox()

	label := canvas.NewText(i18n.T("label_cache_local"), theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	refreshBtn := button.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	progressBar := widgets.NewSizeProgressBar(cache.GetMaxSize(), 0)

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	r := &LocalFilesGuiRenderer{
		g: g,

		Scroll:      container.NewVScroll(list),
		List:        list,
		Label:       label,
		RefreshBtn:  refreshBtn,
		ProgressBar: progressBar,

		itemMap: make(map[string]weak.Pointer[LocalFileGui]),
	}

	r.updateItems()

	go g.RenderLoop()

	return r
}

func (g *LocalFilesGui) RefreshFiles() {
	g.infos = cache.GetLocalCacheInfos()
	fyne.Do(func() {
		g.Refresh()
	})
}

type LocalFilesGuiRenderer struct {
	g *LocalFilesGui

	Scroll      *container.Scroll
	List        *fyne.Container
	Label       *canvas.Text
	RefreshBtn  *button.PaddedIconBtn
	ProgressBar *widgets.SizeProgressBar

	itemMap map[string]weak.Pointer[LocalFileGui]
}

func (r *LocalFilesGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 400)
}

func (r *LocalFilesGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	btnSize := r.RefreshBtn.MinSize().Height
	topHeight := btnSize + p

	labelHeight := r.Label.MinSize().Height

	r.Label.Resize(r.Label.MinSize())
	r.Label.Move(fyne.NewPos(p, (topHeight-labelHeight)/2))
	r.RefreshBtn.Resize(r.RefreshBtn.MinSize())
	r.RefreshBtn.Move(fyne.NewPos(size.Width-btnSize-p/2, p/2))

	progressX := r.Label.MinSize().Width + p*2
	progressWidth := size.Width - progressX - btnSize - p*2
	r.ProgressBar.Resize(fyne.NewSize(progressWidth, btnSize))
	r.ProgressBar.Move(fyne.NewPos(progressX, p/2))

	r.Scroll.Resize(fyne.NewSize(size.Width, size.Height-topHeight-theme.Padding()))
	r.Scroll.Move(fyne.NewPos(0, topHeight+theme.Padding()))
}

func (r *LocalFilesGuiRenderer) updateItems() {
	totalSize := lo.Reduce(r.g.infos, func(sum int64, info types.CacheFileInfo, _ int) int64 {
		return sum + info.Size
	}, 0)

	items := lo.Map(r.g.infos, func(info types.CacheFileInfo, _ int) *LocalFileGui {
		if item, ok := r.itemMap[info.ID]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(info)
				return v
			}
		}
		newGui := NewLocalFileGui(info, false)
		r.itemMap[info.ID] = weak.Make(newGui)
		return newGui
	})

	if r.List.Objects != nil {
		r.List.RemoveAll()
	}
	for _, item := range items {
		r.List.Add(item)
	}
	r.List.Refresh()
	r.Scroll.Refresh()

	r.ProgressBar.SetCurrentSize(totalSize)
}

func (r *LocalFilesGuiRenderer) Refresh() {
	if r.g.changedId != "" {
		if item, ok := r.itemMap[r.g.changedId]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(cache.GetLocalCacheInfo(r.g.changedId))
			}
		}
		r.g.changedId = ""
		return
	}

	r.updateItems()

	canvas.Refresh(r.g)
}

func (r *LocalFilesGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Scroll,
		r.Label,
		r.RefreshBtn,
		r.ProgressBar,
	}
}

func (r *LocalFilesGuiRenderer) Destroy() {
	close(r.g.stopCh)
}

type AllowListGui struct {
	widget.BaseWidget

	infos     []types.CacheFileInfo
	changedId string

	stopCh chan struct{}
}

func NewAllowListGui() *AllowListGui {
	g := &AllowListGui{
		stopCh: make(chan struct{}),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *AllowListGui) RenderLoop() {
	ch := persistence.GetAllowList().SubscribeEvent()
	defer ch.Close()

	for {
		select {
		case <-g.stopCh:
			return
		case message := <-ch.Channel:
			if message[0] == '*' {
				g.changedId = message[1:]
				fyne.Do(func() {
					g.Refresh()
				})
			} else {
				g.RefreshFiles()
			}
		}
	}
}

func (g *AllowListGui) RefreshFiles() {
	g.infos = persistence.GetAllowListEntries()
	fyne.Do(func() {
		g.Refresh()
	})
}

func (g *AllowListGui) CreateRenderer() fyne.WidgetRenderer {
	list := container.NewVBox()

	label := canvas.NewText(i18n.T("label_cache_allow_list"), theme.Color(theme.ColorNameForeground))
	label.TextSize = 14

	refreshBtn := button.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	r := &AllowListGuiRenderer{
		g: g,

		Scroll:     container.NewVScroll(list),
		List:       list,
		Label:      label,
		RefreshBtn: refreshBtn,

		itemMap: make(map[string]weak.Pointer[LocalFileGui]),
	}

	r.updateItems()

	go g.RenderLoop()

	return r
}

type AllowListGuiRenderer struct {
	g *AllowListGui

	Scroll     *container.Scroll
	List       *fyne.Container
	Label      *canvas.Text
	RefreshBtn *button.PaddedIconBtn

	itemMap map[string]weak.Pointer[LocalFileGui]
}

func (r *AllowListGuiRenderer) MinSize() fyne.Size {
	return fyne.NewSize(400, 400)
}

func (r *AllowListGuiRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	btnSize := r.RefreshBtn.MinSize().Height
	topHeight := btnSize + p

	labelHeight := r.Label.MinSize().Height

	r.Label.Resize(r.Label.MinSize())
	r.Label.Move(fyne.NewPos(p, (topHeight-labelHeight)/2))
	r.RefreshBtn.Resize(r.RefreshBtn.MinSize())
	r.RefreshBtn.Move(fyne.NewPos(size.Width-btnSize-p/2, p/2))

	r.Scroll.Resize(fyne.NewSize(size.Width, size.Height-topHeight-theme.Padding()))
	r.Scroll.Move(fyne.NewPos(0, topHeight+theme.Padding()))
}

func (r *AllowListGuiRenderer) updateItems() {
	items := lo.Map(r.g.infos, func(info types.CacheFileInfo, _ int) *LocalFileGui {
		if item, ok := r.itemMap[info.ID]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(info)
				return v
			}
		}
		newGui := NewLocalFileGui(info, true)
		r.itemMap[info.ID] = weak.Make(newGui)
		return newGui
	})

	if r.List.Objects != nil {
		r.List.RemoveAll()
	}
	for _, item := range items {
		r.List.Add(item)
	}
	r.List.Refresh()
	r.Scroll.Refresh()
}

func (r *AllowListGuiRenderer) Refresh() {
	if r.g.changedId != "" {
		if item, ok := r.itemMap[r.g.changedId]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(cache.GetLocalCacheInfo(r.g.changedId))
			}
		}
		r.g.changedId = ""
		return
	}

	r.updateItems()

	canvas.Refresh(r.g)
}

func (r *AllowListGuiRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.Scroll,
		r.Label,
		r.RefreshBtn,
	}
}

func (r *AllowListGuiRenderer) Destroy() {
	close(r.g.stopCh)
}
