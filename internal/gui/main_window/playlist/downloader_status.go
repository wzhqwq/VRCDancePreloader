package playlist

import (
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/host"
)

type DownloaderStatus struct {
	interactive_widgets.LifeCycleWidget

	interval        time.Duration
	intervalChanged bool
}

func NewDownloaderStatus() *DownloaderStatus {
	s := &DownloaderStatus{}
	s.AddLifeCycleFn(s.loop)
	s.ExtendBaseWidget(s)
	return s
}

func (s *DownloaderStatus) CreateRenderer() fyne.WidgetRenderer {
	r := &DownloaderStatusRenderer{
		s: s,

		text: canvas.NewText("", theme.Color(theme.ColorNameWarning)),
	}
	r.Created(s)
	return r
}

func (s *DownloaderStatus) loop(stopCh <-chan struct{}) {
	defaultCh := host.Downloader().SubscribeCoolDownInterval("default")
	defer defaultCh.Close()
	pypyCh := host.Downloader().SubscribeCoolDownInterval("pypy")
	defer pypyCh.Close()

	for {
		select {
		case <-stopCh:
			return
		case interval := <-defaultCh.Channel:
			if interval != s.interval {
				s.interval = interval
				s.intervalChanged = true
				fyne.Do(s.Refresh)
			}
		case interval := <-pypyCh.Channel:
			if interval != s.interval {
				s.interval = interval
				s.intervalChanged = true
				fyne.Do(s.Refresh)
			}
		}
	}
}

type DownloaderStatusRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

	s *DownloaderStatus

	text *canvas.Text
}

func (r *DownloaderStatusRenderer) renderThrottleMessage(seconds float64) {
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

func (r *DownloaderStatusRenderer) Layout(size fyne.Size) {
	r.text.Resize(size)
}

func (r *DownloaderStatusRenderer) MinSize() fyne.Size {
	return r.text.MinSize()
}

func (r *DownloaderStatusRenderer) Refresh() {
	if r.s.intervalChanged {
		r.s.intervalChanged = false
		r.renderThrottleMessage(r.s.interval.Seconds())
	}
	r.text.Refresh()
}

func (r *DownloaderStatusRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.text}
}
