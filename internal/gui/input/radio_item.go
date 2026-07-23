package input

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	_ fyne.Widget       = (*radioItem)(nil)
	_ desktop.Hoverable = (*radioItem)(nil)
	_ fyne.Tappable     = (*radioItem)(nil)
	_ fyne.Focusable    = (*radioItem)(nil)
)

func newRadioItem(label string, onTap func(*radioItem)) *radioItem {
	i := &radioItem{Label: label, onTap: onTap}
	i.ExtendBaseWidget(i)
	return i
}

// radioItem is a single radio item to be used by RadioGroup.
type radioItem struct {
	widget.DisableableWidget

	Label    string
	Selected bool

	focused bool
	hovered bool
	onTap   func(item *radioItem)
}

// CreateRenderer is a private method to Fyne which links this widget to its renderer.
func (i *radioItem) CreateRenderer() fyne.WidgetRenderer {
	r := &radioItemRenderer{item: i}
	r.label.Alignment = fyne.TextAlignLeading
	r.label.TextSize = i.Theme().Size(theme.SizeNameText)
	r.radioOut.StrokeWidth = 2.5

	r.objects = []fyne.CanvasObject{&r.focusIndicator, &r.radioOut, &r.radioIn, &r.label}
	r.update()
	return r
}

// FocusGained is called when this item gained the focus.
func (i *radioItem) FocusGained() {
	i.focused = true
	i.Refresh()
}

// FocusLost is called when this item lost the focus.
func (i *radioItem) FocusLost() {
	i.focused = false
	i.Refresh()
}

// MouseIn is called when a desktop pointer enters the widget.
func (i *radioItem) MouseIn(_ *desktop.MouseEvent) {
	if i.Disabled() {
		return
	}

	i.hovered = true
	i.Refresh()
}

// MouseMoved is called when a desktop pointer hovers over the widget.
func (i *radioItem) MouseMoved(_ *desktop.MouseEvent) {
}

// MouseOut is called when a desktop pointer exits the widget
func (i *radioItem) MouseOut() {
	if i.Disabled() {
		return
	}

	i.hovered = false
	i.Refresh()
}

// SetSelected sets whether this radio item is selected or not.
func (i *radioItem) SetSelected(selected bool) {
	if i.Disabled() || i.Selected == selected {
		return
	}

	i.Selected = selected
	i.Refresh()
}

// Tapped is called when a pointer tapped event is captured and triggers any change handler
func (i *radioItem) Tapped(_ *fyne.PointEvent) {
	if !i.focused {
		if c := fyne.CurrentApp().Driver().CanvasForObject(i); c != nil {
			c.Focus(i)
		}
	}
	i.toggle()
}

// TypedKey is called when this item receives a key event.
func (i *radioItem) TypedKey(_ *fyne.KeyEvent) {
}

// TypedRune is called when this item receives a char event.
func (i *radioItem) TypedRune(r rune) {
	if r == ' ' {
		i.toggle()
	}
}

func (i *radioItem) toggle() {
	if i.Disabled() || i.onTap == nil {
		return
	}

	i.onTap(i)
}

type radioItemRenderer struct {
	objects []fyne.CanvasObject

	item *radioItem

	focusIndicator    canvas.Circle
	radioIn, radioOut canvas.Circle

	label canvas.Text
}

func (r *radioItemRenderer) Layout(size fyne.Size) {
	th := r.item.Theme()
	innerPadding := th.Size(theme.SizeNameInnerPadding)
	borderSize := th.Size(theme.SizeNameInputBorder)
	iconInlineSize := th.Size(theme.SizeNameInlineIcon)

	focusIndicatorSize := fyne.NewSquareSize(iconInlineSize + innerPadding)
	r.focusIndicator.Resize(focusIndicatorSize)
	r.focusIndicator.Move(fyne.NewPos(borderSize, (size.Height-focusIndicatorSize.Height)/2))

	labelSize := fyne.NewSize(size.Width, size.Height)
	r.label.Resize(labelSize)
	r.label.Move(fyne.NewPos(focusIndicatorSize.Width+th.Size(theme.SizeNamePadding), 0))

	iconPos := fyne.NewPos(innerPadding/2+borderSize, (size.Height-iconInlineSize)/2)
	iconSize := fyne.NewSquareSize(iconInlineSize)

	r.radioOut.Move(iconPos)
	r.radioOut.Resize(iconSize)
	r.radioOut.Move(iconPos.AddXY(5, 5))
	r.radioOut.Resize(iconSize.AddWidthHeight(-10, -10))
}

func (r *radioItemRenderer) Destroy() {
}

func (r *radioItemRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *radioItemRenderer) MinSize() fyne.Size {
	th := r.item.Theme()
	inPad := th.Size(theme.SizeNameInnerPadding) * 2

	return r.label.MinSize().
		AddWidthHeight(inPad+th.Size(theme.SizeNameInlineIcon)+th.Size(theme.SizeNamePadding), inPad)
}

func (r *radioItemRenderer) Refresh() {
	r.update()
	canvas.Refresh(r.item)
}

func (r *radioItemRenderer) update() {
	th := r.item.Theme()
	v := fyne.CurrentApp().Settings().ThemeVariant()

	r.label.Text = r.item.Label
	r.label.TextSize = th.Size(theme.SizeNameText)
	if r.item.Disabled() {
		r.label.Color = th.Color(theme.ColorNameDisabled, v)
	} else {
		r.label.Color = th.Color(theme.ColorNameForeground, v)
	}

	if r.item.Disabled() {
		if r.item.Selected {
			r.radioIn.FillColor = theme.Color(theme.ColorNameDisabled)
		} else {
			r.radioIn.FillColor = color.Transparent
		}
		r.radioOut.StrokeColor = theme.Color(theme.ColorNameDisabled)
	} else {
		if r.item.Selected {
			r.radioIn.FillColor = theme.Color(theme.ColorNamePrimary)
			r.radioOut.StrokeColor = theme.Color(theme.ColorNamePrimary)
		} else {
			r.radioIn.FillColor = color.Transparent
			r.radioOut.StrokeColor = theme.Color(theme.ColorNameInputBorder)
		}
	}

	if r.item.Disabled() {
		r.focusIndicator.FillColor = color.Transparent
	} else if r.item.focused {
		r.focusIndicator.FillColor = th.Color(theme.ColorNameFocus, v)
	} else if r.item.hovered {
		r.focusIndicator.FillColor = th.Color(theme.ColorNameHover, v)
	} else {
		r.focusIndicator.FillColor = color.Transparent
	}
}
