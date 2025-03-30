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

func (p *playBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(p.text.MinSize().Width, p.text.MinSize().Height+8)
}

func (p *playBarRenderer) Layout(size fyne.Size) {
	offset := (size.Height-p.MinSize().Height)/2 + 4

	p.rect1.Move(fyne.NewPos(0, offset))
	p.rect2.Move(fyne.NewPos(0, offset))
	p.text.Move(fyne.NewPos(0, offset+4))

	p.rect1.Resize(fyne.NewSize(size.Width, 4))
	p.rect2.Resize(fyne.NewSize(size.Width*p.pb.Progress, 4))
}

func (p *playBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{p.rect1, p.rect2, p.text}
}

func (p *playBarRenderer) Refresh() {
	p.rect2.Resize(fyne.NewSize(p.rect1.Size().Width*p.pb.Progress, 4))
	p.text.Text = p.pb.Text
	p.text.Refresh()
}

func (p *playBarRenderer) Destroy() {
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
