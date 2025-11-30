package playlist

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
)

type DownloaderStatus struct {
	widget.BaseWidget
}

func NewDownloaderStatus() *DownloaderStatus {
	s := &DownloaderStatus{}
	s.ExtendBaseWidget(s)
	return s
}

func (s *DownloaderStatus) CreateRenderer() fyne.WidgetRenderer {
	r := &DownloaderStatusRenderer{
		text:     canvas.NewText("", theme.Color(theme.ColorNameWarning)),
		cancelCh: make(chan struct{}),
	}
	go r.RenderLoop()
	return r
}

type DownloaderStatusRenderer struct {
	text *canvas.Text

	cancelCh chan struct{}
}

func (r *DownloaderStatusRenderer) RenderLoop() {
	ch := download.SubscribeCoolDownInterval()
	for {
		select {
		case interval := <-ch.Channel:
			seconds := interval.Seconds()
			if seconds <= 3 {
				r.text.Text = ""
			} else {
				r.text.Text = i18n.T("message_download_throttled", goeasyi18n.Options{
					Data: map[string]interface{}{
						"Time": strconv.Itoa(int(seconds)),
					},
				})
			}
		}
	}
}

func (r *DownloaderStatusRenderer) Layout(size fyne.Size) {
	r.text.Resize(size)
}

func (r *DownloaderStatusRenderer) MinSize() fyne.Size {
	return r.text.MinSize()
}

func (r *DownloaderStatusRenderer) Refresh() {
	r.text.Refresh()
}

func (r *DownloaderStatusRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.text}
}

func (r *DownloaderStatusRenderer) Destroy() {
	close(r.cancelCh)
}
