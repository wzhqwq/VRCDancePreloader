package button

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type SideActions struct {
	widget.BaseWidget

	Buttons []fyne.CanvasObject

	hovered      bool
	hoverChanged bool
	hoveredIndex int

	gapY     float32
	offsetsY []float32
	left     float32
	right    float32
}

func NewSideActions() *SideActions {
	a := &SideActions{
		hoveredIndex: -1,
	}

	a.ExtendBaseWidget(a)

	return a
}

func (a *SideActions) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewHorizontalGradient(color.Transparent, theme.Color(theme.ColorNameBackground))
	ov := newOverlay()
	ov.onHover = func() {
		a.hovered = true
		a.hoverChanged = true
		a.Refresh()
	}
	ov.onLeave = func() {
		a.hovered = false
		a.hoverChanged = true

		if a.hoveredIndex >= 0 && a.hoveredIndex < len(a.Buttons) {
			if b, ok := a.Buttons[a.hoveredIndex].(desktop.Hoverable); ok {
				b.MouseOut()
			}
		}
		a.hoveredIndex = -1

		a.Refresh()
	}
	ov.onMove = a.MouseMoved

	background.Hide()
	for _, b := range a.Buttons {
		b.Hide()
	}

	return &sideActionsRenderer{
		background: background,

		a:  a,
		ov: ov,
	}
}

func (a *SideActions) getPointingIndex(position fyne.Position) int {
	x, y := position.X, position.Y
	if x < a.left || x > a.right {
		return -1
	}
	if len(a.offsetsY) == 0 || y < a.offsetsY[0] {
		return -1
	}

	for i, b := range a.offsetsY {
		if y < b {
			if y < b-a.gapY {
				return i - 1
			} else {
				return -1
			}
		}
	}
	if y < a.Size().Height-a.gapY {
		return len(a.offsetsY) - 1
	}
	return -1
}

func (a *SideActions) MouseMoved(e *desktop.MouseEvent) {
	pointedIndex := a.getPointingIndex(e.Position)

	var lastButton desktop.Hoverable
	if a.hoveredIndex >= 0 && a.hoveredIndex < len(a.Buttons) {
		if b, ok := a.Buttons[a.hoveredIndex].(desktop.Hoverable); ok {
			lastButton = b
		}
	}

	if pointedIndex == -1 {
		if lastButton != nil {
			lastButton.MouseOut()
		}
	} else {
		ev := &desktop.MouseEvent{Button: e.Button}
		ev.AbsolutePosition = e.AbsolutePosition
		ev.Position = fyne.NewPos(e.Position.X-a.left, e.Position.Y-a.offsetsY[pointedIndex])

		if lastButton != nil {
			if a.hoveredIndex == pointedIndex {
				lastButton.MouseMoved(ev)
				return
			} else {
				lastButton.MouseOut()
			}
		}

		if pointedIndex < len(a.Buttons) {
			if b, ok := a.Buttons[pointedIndex].(desktop.Hoverable); ok {
				b.MouseIn(ev)
			}
		}
	}

	a.hoveredIndex = pointedIndex
}

type sideActionsRenderer struct {
	background fyne.CanvasObject
	ov         *overlay

	a *SideActions
}

func (r *sideActionsRenderer) MinSize() fyne.Size {
	itemWidth := float32(0)
	if r.a.Buttons != nil {
		itemWidth = r.a.Buttons[0].MinSize().Width
	}
	return fyne.NewSize(itemWidth*2+10, 100)
}

func (r *sideActionsRenderer) Layout(size fyne.Size) {
	itemWidth := float32(0)
	if r.a.Buttons != nil {
		itemWidth = r.a.Buttons[0].MinSize().Width
	}
	containerWidth := itemWidth*2 + 10
	containerOffset := size.Width - containerWidth

	r.ov.Resize(size)
	r.ov.Move(fyne.NewPos(0, 0))

	r.background.Resize(fyne.NewSize(containerWidth, size.Height))
	r.background.Move(fyne.NewPos(containerOffset, 0))

	heightSum := float32(0)
	for _, b := range r.a.Buttons {
		b.Resize(b.MinSize())
		heightSum += b.MinSize().Height
	}

	gap := (size.Height - heightSum) / (float32)(len(r.a.Buttons)+1)
	offset := gap
	var offsets []float32
	for _, b := range r.a.Buttons {
		offsets = append(offsets, offset)
		b.Move(fyne.NewPos(containerOffset+itemWidth, offset))
		offset += b.MinSize().Width + gap
	}

	r.a.gapY = gap
	r.a.offsetsY = offsets
	r.a.left = containerOffset + itemWidth
	r.a.right = size.Width - 10
}

func (r *sideActionsRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.background}
	for _, b := range r.a.Buttons {
		objects = append(objects, b)
	}
	objects = append(objects, r.ov)
	return objects
}

func (r *sideActionsRenderer) Refresh() {
	if r.a.hoverChanged {
		r.a.hoverChanged = false
		if r.a.hovered {
			r.background.Show()
			for _, b := range r.a.Buttons {
				b.Show()
			}
		} else {
			r.background.Hide()
			for _, b := range r.a.Buttons {
				b.Hide()
			}
		}
	}

	canvas.Refresh(r.a)
}

func (r *sideActionsRenderer) Destroy() {
}

type overlay struct {
	widget.BaseWidget
	desktop.Hoverable

	onHover func()
	onLeave func()
	onMove  func(*desktop.MouseEvent)
}

func newOverlay() *overlay {
	o := &overlay{}

	o.ExtendBaseWidget(o)

	return o
}

func (o *overlay) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (o *overlay) MouseIn(_ *desktop.MouseEvent) {
	if o.onHover != nil {
		o.onHover()
	}
}
func (o *overlay) MouseOut() {
	if o.onLeave != nil {
		o.onLeave()
	}
}
func (o *overlay) MouseMoved(e *desktop.MouseEvent) {
	o.onMove(e)
}
