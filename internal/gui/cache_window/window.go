package cache_window

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/cache/video_cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
)

var openedWindow fyne.Window

func OpenCacheWindow() {
	if openedWindow != nil {
		return
	}

	openedWindow = custom_fyne.NewWindow(i18n.T("label_cache_local"))

	cacheWindow := &CacheWindow{
		loadedCh: make(chan struct{}),
	}
	cacheWindow.ExtendBaseWidget(cacheWindow)

	openedWindow.SetContent(cacheWindow)
	openedWindow.SetPadded(false)
	openedWindow.Show()
	openedWindow.SetOnClosed(func() {
		openedWindow = nil
	})

	cacheWindow.loadedCh <- struct{}{}
}

type CacheWindow struct {
	widget.BaseWidget

	loadedCh chan struct{}
}

func (c *CacheWindow) CreateRenderer() fyne.WidgetRenderer {
	localFiles := NewFileListGui(false)
	preserved := NewFileListGui(true)
	divider := canvas.NewRectangle(theme.Color(theme.ColorNameSeparator))
	progressBar := widgets.NewSizeProgressBar(video_cache.GetMaxSize(), 0)

	r := &cacheWindowRenderer{
		c: c,

		localFiles:  localFiles,
		preserved:   preserved,
		divider:     divider,
		progressBar: progressBar,

		stopCh: make(chan struct{}, 1),
	}

	go func() {
		<-c.loadedCh
		<-time.After(time.Millisecond * 300)
		r.updateTotalSize()
		localFiles.loadedCh <- struct{}{}
		preserved.loadedCh <- struct{}{}

		r.Loop()
	}()

	return r
}

type cacheWindowRenderer struct {
	c *CacheWindow

	localFiles  *FileListGui
	preserved   *FileListGui
	progressBar *widgets.SizeProgressBar
	divider     *canvas.Rectangle

	stopCh chan struct{}
}

func (r *cacheWindowRenderer) Loop() {
	ch := persistence.SubscribeMetaTableChange()
	defer ch.Close()
	for {
		select {
		case e := <-ch.Channel:
			if e.Type == "video" && e.Op != '*' {
				r.updateTotalSize()
			}
		case <-r.stopCh:
			return
		}
	}
}

func (r *cacheWindowRenderer) Destroy() {
	close(r.stopCh)
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
	r.progressBar.Refresh()
	r.localFiles.Refresh()
	r.preserved.Refresh()
}

func (r *cacheWindowRenderer) updateTotalSize() {
	totalSize, err := persistence.SummarizeCacheSize()
	if err != nil {
		return
	}

	size, ok := totalSize["video"]
	fyne.Do(func() {
		if ok {
			r.progressBar.SetCurrentSize(size)
		} else {
			r.progressBar.SetCurrentSize(0)
		}
	})
}
