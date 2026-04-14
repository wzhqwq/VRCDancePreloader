package widgets

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DownloadBar struct {
	widget.BaseWidget
	Progress     float32
	ProgressText string
	Speed        string
	Remaining    string
}

type downloadBarRenderer struct {
	rect1 *canvas.Rectangle
	rect2 *canvas.Rectangle

	progress  *canvas.Text
	speed     *canvas.Text
	remaining *canvas.Text

	pb *DownloadBar
}

func (r *downloadBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.progress.MinSize().Width+r.speed.MinSize().Width, r.progress.MinSize().Height+8)
}

func (r *downloadBarRenderer) layoutText(size fyne.Size) {
	offset := (size.Height - r.MinSize().Height) / 2

	progressRight := r.progress.MinSize().Width + 4
	speedWidth := r.speed.MinSize().Width
	remainingWidth := r.remaining.MinSize().Width

	r.progress.Resize(r.progress.MinSize())
	r.speed.Resize(r.speed.MinSize())
	r.remaining.Resize(r.remaining.MinSize())

	r.progress.Move(fyne.NewPos(0, offset))
	if progressRight+speedWidth+remainingWidth > size.Width {
		r.remaining.Hide()
		if progressRight+speedWidth > size.Width {
			r.speed.Hide()
		} else {
			r.speed.Move(fyne.NewPos(size.Width-speedWidth, offset))
			r.speed.Show()
		}
	} else {
		r.speed.Move(fyne.NewPos(size.Width-speedWidth-remainingWidth, offset))
		r.remaining.Move(fyne.NewPos(size.Width-remainingWidth, offset))
		r.speed.Show()
		r.remaining.Show()
	}
}

func (r *downloadBarRenderer) Layout(size fyne.Size) {
	offset := (size.Height - r.MinSize().Height) / 2

	r.layoutText(size)

	r.rect1.Move(fyne.NewPos(0, size.Height-offset-4))
	r.rect2.Move(fyne.NewPos(0, size.Height-offset-4))

	r.rect1.Resize(fyne.NewSize(size.Width, 4))
	r.rect2.Resize(fyne.NewSize(size.Width*r.pb.Progress, 4))
}

func (r *downloadBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect1, r.rect2, r.progress, r.speed, r.remaining}
}

func (r *downloadBarRenderer) Refresh() {
	r.progress.Text = r.pb.ProgressText
	r.speed.Text = r.pb.Speed
	r.remaining.Text = r.pb.Remaining

	size := r.pb.Size()
	r.layoutText(size)
	r.rect2.Resize(fyne.NewSize(size.Width*r.pb.Progress, 4))

	canvas.Refresh(r.pb)
}

func (r *downloadBarRenderer) Destroy() {
}

func NewDownloadBar() *DownloadBar {
	p := &DownloadBar{}
	p.ExtendBaseWidget(p)
	return p
}

func (p *DownloadBar) SetProgress(total, downloaded int64, speed float64, remaining time.Duration) {
	p.Progress = float32(downloaded) / float32(total)
	p.ProgressText = utils.PrettyByteSize(downloaded) + "/" + utils.PrettyByteSize(total)
	p.Speed = utils.PrettyByteSizeF(speed) + "/s"
	if remaining > 0 {
		p.Remaining = " (" + i18n.T("message_time_remaining", goeasyi18n.Options{
			Data: map[string]any{
				"Duration": i18n.ParseDuration(remaining),
			},
		}) + ")"
	} else {
		p.Remaining = ""
	}
	p.Refresh()
}

func (p *DownloadBar) CreateRenderer() fyne.WidgetRenderer {
	rect1 := canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground))
	rect1.CornerRadius = 2
	rect2 := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	rect2.CornerRadius = 2

	text1 := canvas.NewText(p.ProgressText, theme.Color(theme.ColorNameForeground))
	text1.TextSize = 12
	text2 := canvas.NewText(p.Speed, theme.Color(theme.ColorNameForeground))
	text2.TextSize = 12
	text3 := canvas.NewText(p.ProgressText, theme.Color(theme.ColorNameForeground))
	text3.TextSize = 12

	return &downloadBarRenderer{
		rect1: rect1,
		rect2: rect2,

		progress:  text1,
		speed:     text2,
		remaining: text3,

		pb: p,
	}
}
