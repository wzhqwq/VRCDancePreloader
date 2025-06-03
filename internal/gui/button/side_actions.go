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

	hovered bool
}

func NewSideActions() *SideActions {
	a := &SideActions{}

	a.ExtendBaseWidget(a)

	return a
}

func (a *SideActions) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewHorizontalGradient(color.Transparent, theme.Color(theme.ColorNameBackground))
	ov := newOverlay()
	ov.onHover = func() {
		a.hovered = true
		a.Refresh()
	}
	ov.onLeave = func() {
		a.hovered = false
		a.Refresh()
	}

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
	for _, b := range r.a.Buttons {
		b.Move(fyne.NewPos(containerOffset+itemWidth, offset))
		offset += b.MinSize().Width + gap
	}
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

	canvas.Refresh(r.a)
}

func (r *sideActionsRenderer) Destroy() {
}

type overlay struct {
	widget.BaseWidget
	desktop.Hoverable

	onHover func()
	onLeave func()
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
func (o *overlay) MouseMoved(_ *desktop.MouseEvent) {
}
