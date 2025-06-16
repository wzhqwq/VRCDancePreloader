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

func (r *ellipseTextRenderer) MinSize() fyne.Size {
	return fyne.NewSize(50, r.text.MinSize().Height)
}

func (r *ellipseTextRenderer) Layout(size fyne.Size) {
	r.width = size.Width
	r.text.Resize(size)
	r.text.Text = r.findProperSlice()
}

func (r *ellipseTextRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.text}
}

func (r *ellipseTextRenderer) Refresh() {
	r.text.Text = r.findProperSlice()
	r.text.TextSize = r.e.TextSize
	r.text.TextStyle = r.e.TextStyle
	r.text.Color = r.e.Color
}

func (r *ellipseTextRenderer) Destroy() {
}

func (r *ellipseTextRenderer) findProperSlice() string {
	full := r.e.Text
	if r.calculateSize(full) <= r.width {
		return full
	}

	ellipsis := "..."
	ellipsisWidth := r.calculateSize(ellipsis)
	runes := []rune(full)
	n := len(runes)

	low, high := 0, n
	for low < high {
		mid := (low + high + 1) / 2
		slice := string(runes[:mid])
		if r.calculateSize(slice)+ellipsisWidth <= r.width {
			low = mid
		} else {
			high = mid - 1
		}
	}

	if low <= 0 {
		if ellipsisWidth <= r.width {
			return ellipsis
		}
		return ""
	}

	return string(runes[:low]) + ellipsis
}
func (r *ellipseTextRenderer) calculateSize(text string) float32 {
	size, _ := window_app.Driver().RenderedTextSize(text, r.e.TextSize, r.e.TextStyle, nil)
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
