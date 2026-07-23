package cache_window

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
)

var openedWindow fyne.Window

func OpenCacheWindow() {
	if openedWindow != nil {
		return
	}

	openedWindow = fyne.CurrentApp().NewWindow(i18n.T("label_cache_local"))

	cacheWindow := &CacheWindow{}
	cacheWindow.ExtendBaseWidget(cacheWindow)
	cacheWindow.AddLifeCycleFn(cacheWindow.loop)

	openedWindow.SetContent(cacheWindow)
	openedWindow.SetPadded(false)
	openedWindow.Show()
	openedWindow.SetOnClosed(func() {
		openedWindow = nil
	})
}

type CacheWindow struct {
	interactive_widgets.LifeCycleWidget

	space int64

	spaceChanged bool
}

func (c *CacheWindow) loop(stopCh <-chan struct{}) {
	ch := persistence.SubscribeTotalSizeChange()
	defer ch.Close()

	c.updateTotalSize()

	for {
		select {
		case e := <-ch.Channel:
			if e.Type == "video" {
				c.totalSizeDelta(e.Delta)
			}
		case <-stopCh:
			return
		}
	}
}

func (c *CacheWindow) updateTotalSize() {
	totalSize, err := persistence.SummarizeCacheSize()
	if err != nil {
		return
	}

	if size, ok := totalSize["video"]; ok {
		c.space = size
	} else {
		c.space = 0
	}
	c.spaceChanged = true

	fyne.Do(func() {
		c.Refresh()
	})
}

func (c *CacheWindow) totalSizeDelta(delta int64) {
	c.space += delta
	c.spaceChanged = true
	fyne.Do(func() {
		c.Refresh()
	})
}

func (c *CacheWindow) CreateRenderer() fyne.WidgetRenderer {
	localFiles := NewFileListGui(false)
	preserved := NewFileListGui(true)
	divider := canvas.NewRectangle(theme.Color(theme.ColorNameSeparator))
	progressBar := widgets.NewSizeProgressBar(host.CacheManager().CacheSize(), 0)

	r := &cacheWindowRenderer{
		c: c,

		localFiles:  localFiles,
		preserved:   preserved,
		divider:     divider,
		progressBar: progressBar,
	}
	r.Created(c)

	return r
}

type cacheWindowRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	c *CacheWindow

	localFiles  *FileListGui
	preserved   *FileListGui
	progressBar *widgets.SizeProgressBar
	divider     *canvas.Rectangle
}

func (r *cacheWindowRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	topHeight := r.progressBar.MinSize().Height
	r.progressBar.Resize(fyne.NewSize(size.Width-p*2, topHeight))
	r.progressBar.Move(fyne.NewPos(p, p))
	topHeight += p * 2

	r.localFiles.Resize(fyne.NewSize(size.Width/2, size.Height-topHeight))
	r.localFiles.Move(fyne.NewPos(0, topHeight))

	r.preserved.Resize(fyne.NewSize(size.Width/2, size.Height-topHeight))
	r.preserved.Move(fyne.NewPos(size.Width/2, topHeight))

	r.divider.Resize(fyne.NewSize(1, size.Height-topHeight))
	r.divider.Move(fyne.NewPos(size.Width/2, topHeight))
}

func (r *cacheWindowRenderer) MinSize() fyne.Size {
	return fyne.NewSize(600, 300)
}

func (r *cacheWindowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.localFiles, r.divider, r.preserved, r.progressBar}
}

func (r *cacheWindowRenderer) Refresh() {
	if r.c.spaceChanged {
		r.progressBar.SetCurrentSize(r.c.space)
		r.c.spaceChanged = false
		r.progressBar.Refresh()
	}

	r.localFiles.Refresh()
	r.preserved.Refresh()
}
