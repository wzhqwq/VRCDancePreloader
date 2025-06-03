package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
)

type Rate struct {
	widget.BaseWidget
	fyne.Tabbable

	Score int
	Type  string
	label string

	OnChanged func(score int)
}

func NewRate(score int, label, iconType string) *Rate {
	r := &Rate{
		Type:  iconType,
		label: label,
		Score: score,
	}

	r.ExtendBaseWidget(r)

	return r
}

func (r *Rate) SetScore(score int) {
	r.Score = score
	fyne.Do(func() {
		r.Refresh()
	})
}

func (r *Rate) Tapped(e *fyne.PointEvent) {
	position := int((e.Position.X - 60) / 24)
	if position > 4 {
		return
	}
	newScore := position + 1
	if newScore != r.Score {
		r.SetScore(newScore)
		if r.OnChanged != nil {
			r.OnChanged(newScore)
		}
	}
}

func (r *Rate) CreateRenderer() fyne.WidgetRenderer {
	labelWidget := canvas.NewText(r.label, theme.Color(theme.ColorNamePlaceHolder))

	iconList := make([]*widget.Icon, 5)
	for i := 0; i < 5; i++ {
		iconList[i] = widget.NewIcon(icons.GetIcon(r.Type))
	}
	return &rateRenderer{
		r: r,

		Label: labelWidget,
		Icons: iconList,
	}
}

type rateRenderer struct {
	r *Rate

	Label *canvas.Text
	Icons []*widget.Icon
}

func (r *rateRenderer) MinSize() fyne.Size {
	return fyne.NewSize(180, 24)
}

func (r *rateRenderer) Layout(size fyne.Size) {
	iconSize := fyne.NewSize(18, 18)
	labelSize := r.Label.MinSize()

	r.Label.Resize(labelSize)
	r.Label.Move(fyne.NewPos(56-labelSize.Width, (24-labelSize.Height)/2))
	for i := 0; i < 5; i++ {
		r.Icons[i].Resize(iconSize)
		r.Icons[i].Move(fyne.NewPos(float32(63+i*24), 3))
	}
}

func (r *rateRenderer) Objects() []fyne.CanvasObject {
	objects := make([]fyne.CanvasObject, 6)
	objects[0] = r.Label
	for i := 1; i < 6; i++ {
		objects[i] = r.Icons[i-1]
	}
	return objects
}

func (r *rateRenderer) Refresh() {
	for i := 0; i < 5; i++ {
		if i < r.r.Score {
			r.Icons[i].SetResource(icons.GetIcon(r.r.Type + "-fill"))
		} else {
			r.Icons[i].SetResource(icons.GetIcon(r.r.Type))
		}
	}
}

func (r *rateRenderer) Destroy() {
}
