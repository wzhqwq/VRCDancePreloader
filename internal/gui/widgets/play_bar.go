package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type PlayBar struct {
	widget.BaseWidget
	Progress float32
	Text     string
}

type playBarRenderer struct {
	rect1 *canvas.Rectangle
	rect2 *canvas.Rectangle
	text  *canvas.Text

	pb *PlayBar
}

func (r *playBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.text.MinSize().Width, r.text.MinSize().Height+8)
}

func (r *playBarRenderer) Layout(size fyne.Size) {
	offset := (size.Height-r.MinSize().Height)/2 + 4

	r.rect1.Move(fyne.NewPos(0, offset))
	r.rect2.Move(fyne.NewPos(0, offset))
	r.text.Move(fyne.NewPos(0, offset+4))

	r.rect1.Resize(fyne.NewSize(size.Width, 4))
	r.rect2.Resize(fyne.NewSize(size.Width*r.pb.Progress, 4))
}

func (r *playBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.rect1, r.rect2, r.text}
}

func (r *playBarRenderer) Refresh() {
	r.text.Text = r.pb.Text
	r.rect2.Resize(fyne.NewSize(r.pb.Size().Width*r.pb.Progress, 4))
	canvas.Refresh(r.pb)
}

func (r *playBarRenderer) Destroy() {
}

func NewPlayBar() *PlayBar {
	p := &PlayBar{}
	p.ExtendBaseWidget(p)
	return p
}

func (p *PlayBar) CreateRenderer() fyne.WidgetRenderer {
	rect1 := canvas.NewRectangle(color.Gray{Y: 200})
	rect1.CornerRadius = 2
	rect2 := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	rect2.CornerRadius = 2
	text := canvas.NewText(p.Text, theme.Color(theme.ColorNamePrimary))
	text.TextSize = 12

	return &playBarRenderer{
		rect1: rect1,
		rect2: rect2,
		text:  text,

		pb: p,
	}
}
