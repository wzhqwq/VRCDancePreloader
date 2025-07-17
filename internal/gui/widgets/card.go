package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type Card struct {
	widget.BaseWidget

	Content fyne.CanvasObject
}

func (c *Card) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.White)
	rect.CornerRadius = theme.Padding() * 2
	return &CardRenderer{
		rect: rect,
		c:    c,
	}
}

func NewCard(content fyne.CanvasObject) *Card {
	c := &Card{
		Content: content,
	}
	c.ExtendBaseWidget(c)
	return c
}

type CardRenderer struct {
	rect *canvas.Rectangle
	c    *Card
}

func (c CardRenderer) Layout(size fyne.Size) {
	p := theme.Padding()
	c.rect.Resize(size)
	c.rect.Move(fyne.NewPos(0, 0))
	c.c.Content.Resize(fyne.NewSize(size.Width-p*2, size.Height-p*2))
	c.c.Content.Move(fyne.NewPos(p, p))
}

func (c CardRenderer) MinSize() fyne.Size {
	p := theme.Padding()
	return fyne.NewSize(c.c.Content.MinSize().Width+p*2, c.c.Content.MinSize().Height+p*2)
}

func (c CardRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{c.rect, c.c.Content}
}

func (c CardRenderer) Refresh() {
	canvas.Refresh(c.c)
}

func (c CardRenderer) Destroy() {
}
