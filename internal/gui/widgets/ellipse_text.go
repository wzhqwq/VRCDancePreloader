package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/window_app"
)

type EllipseText struct {
	widget.BaseWidget
	Text      string
	TextSize  float32
	TextStyle fyne.TextStyle
	Color     color.Color
}

type ellipseTextRenderer struct {
	text *canvas.Text

	width float32

	e *EllipseText
}

func (p *ellipseTextRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, p.text.MinSize().Height)
}

func (p *ellipseTextRenderer) Layout(size fyne.Size) {
	p.width = size.Width
	p.text.Resize(size)
	p.text.Text = p.findProperSlice()
	p.text.Refresh()
}

func (p *ellipseTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{p.text}
}

func (p *ellipseTextRenderer) Refresh() {
	p.text.Text = p.findProperSlice()
	p.text.TextSize = p.e.TextSize
	p.text.TextStyle = p.e.TextStyle
	p.text.Color = p.e.Color
	p.text.Refresh()
}

func (p *ellipseTextRenderer) Destroy() {
}

func (p *ellipseTextRenderer) findProperSlice() string {
	if p.width < p.e.TextSize {
		return ""
	}
	maxWidth := p.width
	originalText := []rune(p.e.Text)
	if p.calculateSize(string(originalText)) <= maxWidth {
		return string(originalText)
	}
	text := originalText
	iter := 0
	for {
		width := p.calculateSize(string(text) + "...")
		estimatedLen := int(float32(len(text)+3)*(maxWidth/width)) - 3
		if width <= maxWidth {
			if estimatedLen >= len(text)-1 || iter > 10 {
				return string(text) + "..."
			}
			text = originalText[:estimatedLen]
		}
		if width > maxWidth {
			if estimatedLen <= 0 {
				return ""
			}
			if estimatedLen == len(text) {
				estimatedLen--
			}
			text = originalText[:estimatedLen]
		}
		iter++
	}
}
func (p *ellipseTextRenderer) calculateSize(text string) float32 {
	size, _ := window_app.Driver().RenderedTextSize(text, p.e.TextSize, p.e.TextStyle, nil)
	return size.Width
}

func (e *EllipseText) CreateRenderer() fyne.WidgetRenderer {
	e.ExtendBaseWidget(e)
	text := canvas.NewText(e.Text, e.Color)
	text.TextSize = e.TextSize
	text.TextStyle = e.TextStyle
	return &ellipseTextRenderer{
		text:  text,
		width: text.MinSize().Width,
		e:     e,
	}
}

func NewEllipseText(text string, color color.Color) *EllipseText {
	return &EllipseText{
		Text:     text,
		TextSize: theme.TextSize(),
		Color:    color,
	}
}
