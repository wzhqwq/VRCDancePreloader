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
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"sync"
	"weak"
)

type LocalFilesGui struct {
	widget.BaseWidget

	localFileUpdate *utils.StringEventSubscriber
	allowListUpdate *utils.StringEventSubscriber

	infos     []types.CacheFileInfo
	changedId string
}

func NewLocalFilesGui() *LocalFilesGui {
	g := &LocalFilesGui{
		localFileUpdate: cache.SubscribeLocalFileEvent(),
		allowListUpdate: persistence.GetAllowList().SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *LocalFilesGui) RenderLoop() {
	for {
		select {
		case message, ok := <-g.localFileUpdate.Channel:
			if !ok {
				return
			}
			if message[0] == '*' {
				g.changedId = message[1:]
				fyne.Do(func() {
					g.Refresh()
				})
			} else {
				g.RefreshFiles()
			}
		case message, ok := <-g.allowListUpdate.Channel:
			if !ok {
				return
			}
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

	refreshBtn := widgets.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	progressBar := widgets.NewSizeProgressBar(cache.GetMaxSize(), 0)

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	go g.RenderLoop()

	return &LocalFilesGuiRenderer{
		g: g,

		Scroll:      container.NewVScroll(list),
		List:        list,
		Label:       label,
		RefreshBtn:  refreshBtn,
		ProgressBar: progressBar,

		itemMap: make(map[string]weak.Pointer[LocalFileGui]),
	}
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
	RefreshBtn  *widgets.PaddedIconBtn
	ProgressBar *widgets.SizeProgressBar

	itemMap  map[string]weak.Pointer[LocalFileGui]
	mapMutex sync.Mutex
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

func (r *LocalFilesGuiRenderer) Refresh() {
	if r.g.changedId != "" {
		r.mapMutex.Lock()
		if item, ok := r.itemMap[r.g.changedId]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(cache.GetLocalCacheInfo(r.g.changedId))
			}
		}
		r.mapMutex.Unlock()
		r.g.changedId = ""
		return
	}

	totalSize := int64(0)

	var items []*LocalFileGui
	r.mapMutex.Lock()
	for _, info := range r.g.infos {
		totalSize += info.Size

		if item, ok := r.itemMap[info.ID]; ok {
			if v := item.Value(); v != nil {
				items = append(items, v)
				continue
			}
		}
		newGui := NewLocalFileGui(info, false)
		r.itemMap[info.ID] = weak.Make(newGui)
		items = append(items, newGui)
	}
	r.mapMutex.Unlock()

	r.List.RemoveAll()
	for _, item := range items {
		r.List.Add(item)
	}
	r.List.Refresh()

	r.ProgressBar.SetCurrentSize(totalSize)

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
	if r.g.localFileUpdate != nil {
		r.g.localFileUpdate.Close()
	}
	if r.g.allowListUpdate != nil {
		r.g.allowListUpdate.Close()
	}
}

type AllowListGui struct {
	widget.BaseWidget

	listUpdate *utils.StringEventSubscriber

	infos     []types.CacheFileInfo
	changedId string
}

func NewAllowListGui() *AllowListGui {
	g := &AllowListGui{
		listUpdate: persistence.GetAllowList().SubscribeEvent(),
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *AllowListGui) RenderLoop() {
	for {
		select {
		case message, ok := <-g.listUpdate.Channel:
			if !ok {
				return
			}
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

	refreshBtn := widgets.NewPaddedIconBtn(theme.ViewRefreshIcon())
	refreshBtn.SetMinSquareSize(30)

	refreshBtn.OnClick = func() {
		g.RefreshFiles()
	}

	go g.RenderLoop()

	return &AllowListGuiRenderer{
		g: g,

		Scroll:     container.NewVScroll(list),
		List:       list,
		Label:      label,
		RefreshBtn: refreshBtn,

		itemMap: make(map[string]weak.Pointer[LocalFileGui]),
	}
}

type AllowListGuiRenderer struct {
	g *AllowListGui

	Scroll     *container.Scroll
	List       *fyne.Container
	Label      *canvas.Text
	RefreshBtn *widgets.PaddedIconBtn

	itemMap  map[string]weak.Pointer[LocalFileGui]
	mapMutex sync.Mutex
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

func (r *AllowListGuiRenderer) Refresh() {
	if r.g.changedId != "" {
		r.mapMutex.Lock()
		if item, ok := r.itemMap[r.g.changedId]; ok {
			if v := item.Value(); v != nil {
				v.UpdateInfo(cache.GetLocalCacheInfo(r.g.changedId))
			}
		}
		r.mapMutex.Unlock()
		r.g.changedId = ""
		return
	}

	var items []*LocalFileGui
	r.mapMutex.Lock()
	for _, info := range r.g.infos {
		if item, ok := r.itemMap[info.ID]; ok {
			if v := item.Value(); v != nil {
				items = append(items, v)
				continue
			}
		}
		newGui := NewLocalFileGui(info, true)
		r.itemMap[info.ID] = weak.Make(newGui)
		items = append(items, newGui)
	}
	r.mapMutex.Unlock()

	r.List.RemoveAll()
	for _, item := range items {
		r.List.Add(item)
	}
	r.List.Refresh()

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
	if r.g.listUpdate != nil {
		r.g.listUpdate.Close()
	}
}
