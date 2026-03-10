package scroll

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
)

type Orientation int

const (
	Vertical Orientation = iota
	Horizontal
)

type Bar struct {
	widget.BaseWidget
	desktop.Hoverable
	desktop.Mouseable
	fyne.Draggable

	contentLength float32
	viewLength    float32
	offset        float32

	thumbOffset float32
	thumbLength float32

	holding bool
	hovered bool

	startPos float32

	offsetChangeFn func(float32)

	orientation Orientation
}

func NewBar(orientation Orientation, onOffsetChange func(float32)) *Bar {
	b := &Bar{
		orientation:    orientation,
		offsetChangeFn: onOffsetChange,
	}
	b.ExtendBaseWidget(b)
	return b
}

func (b *Bar) MouseDown(e *desktop.MouseEvent) {
	pos := e.Position.X
	if b.orientation == Vertical {
		pos = e.Position.Y
	}

	if b.thumbOffset <= pos && pos <= b.thumbOffset+b.thumbLength {
		b.holding = true
		b.Refresh()
	}
}
func (b *Bar) MouseUp(e *desktop.MouseEvent) {
	if b.holding {
		b.holding = false
		b.Refresh()
	} else {
		pos := e.Position.X
		if b.orientation == Vertical {
			pos = e.Position.Y
		}

		b.thumbOffset = max(0, min(1, pos/b.viewLength)) * (b.viewLength - b.thumbLength)
		b.offset = b.thumbOffset * (b.contentLength - b.viewLength) / (b.viewLength - b.thumbLength)
		if b.offsetChangeFn != nil {
			b.offsetChangeFn(b.offset)
		}
		b.Refresh()
	}
}

func (b *Bar) MouseIn(_ *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}
func (b *Bar) MouseMoved(_ *desktop.MouseEvent) {
}
func (b *Bar) MouseOut() {
	b.hovered = false
	b.Refresh()
}

func (b *Bar) DragEnd() {
	b.holding = false
	b.Refresh()
}
func (b *Bar) Dragged(e *fyne.DragEvent) {
	if b.holding {
		delta := e.Dragged.DX
		if b.orientation == Vertical {
			delta = e.Dragged.DY
		}

		b.thumbOffset = max(0, min(b.viewLength-b.thumbLength, b.thumbOffset+delta))
		b.offset = b.thumbOffset * (b.contentLength - b.viewLength) / (b.viewLength - b.thumbLength)
		if b.offsetChangeFn != nil {
			b.offsetChangeFn(b.offset)
		}

		b.Refresh()
	}
}

func (b *Bar) calculateThumb() {
	if b.contentLength-b.viewLength < 0.5 {
		b.thumbOffset = 0
		b.thumbLength = 0
	} else {
		b.thumbLength = max(10, b.viewLength*b.viewLength/b.contentLength)
		b.thumbOffset = b.offset * (b.viewLength - b.thumbLength) / (b.contentLength - b.viewLength)
	}
}

func (b *Bar) SetOffset(offset float32) {
	b.offset = offset
	b.calculateThumb()
	b.Refresh()
}

func (b *Bar) SetContentLength(length float32) {
	b.contentLength = length
	b.calculateThumb()
	b.Refresh()
}

const trackSize = 14

func (b *Bar) CreateRenderer() fyne.WidgetRenderer {
	track := canvas.NewRectangle(color.Transparent)
	thumb := canvas.NewRectangle(theme.Color(custom_fyne.ColorNameScrollThumbColor))

	return &barRenderer{
		bar:   b,
		track: track,
		thumb: thumb,
	}
}

type barRenderer struct {
	bar *Bar

	track *canvas.Rectangle
	thumb *canvas.Rectangle
}

func (r *barRenderer) Destroy() {
}

func (r *barRenderer) layout(size fyne.Size) {
	r.track.Move(fyne.NewPos(0, 0))
	r.track.Resize(size)

	thumbSize := float32(2)
	if r.bar.holding {
		thumbSize = float32(10)
	} else if r.bar.hovered {
		thumbSize = float32(8)
	}
	thumbMargin := (trackSize - thumbSize) / 2
	thumbRadius := thumbSize / 2

	r.thumb.CornerRadius = thumbRadius
	if r.bar.orientation == Horizontal {
		r.thumb.Move(fyne.NewPos(thumbMargin+r.bar.thumbOffset, thumbMargin))
		r.thumb.Resize(fyne.NewSize(r.bar.thumbLength-thumbMargin*2, thumbSize))
	} else {
		r.thumb.Move(fyne.NewPos(thumbMargin, thumbMargin+r.bar.thumbOffset))
		r.thumb.Resize(fyne.NewSize(thumbSize, r.bar.thumbLength-thumbMargin*2))
	}
}

func (r *barRenderer) Layout(size fyne.Size) {
	r.bar.viewLength = size.Height
	if r.bar.orientation == Horizontal {
		r.bar.viewLength = size.Width
	}
	r.bar.calculateThumb()

	r.layout(size)
}

func (r *barRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(trackSize)
}

func (r *barRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.thumb}
}

func (r *barRenderer) Refresh() {
	r.layout(r.bar.Size())

	if r.bar.holding {
		r.thumb.FillColor = theme.Color(custom_fyne.ColorNameScrollThumbActive)
		r.track.FillColor = theme.Color(custom_fyne.ColorNameScrollTrackHover)
	} else {
		r.thumb.FillColor = theme.Color(custom_fyne.ColorNameScrollThumbColor)
		if r.bar.hovered {
			r.track.FillColor = theme.Color(custom_fyne.ColorNameScrollTrackHover)
		} else {
			r.track.FillColor = color.Transparent
		}
	}
}
