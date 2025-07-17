package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
)

type IconWithMessage struct {
	widget.BaseWidget
	desktop.Hoverable
	desktop.Cursorable

	Icon    *widget.Icon
	Message *canvas.Text
}

func NewIconWithMessage(icon fyne.Resource) *IconWithMessage {
	t := &IconWithMessage{
		Icon:    widget.NewIcon(icon),
		Message: canvas.NewText("", theme.Color(theme.ColorNameForeground)),
	}
	t.Message.TextSize = 12
	t.Message.Hide()
	t.ExtendBaseWidget(t)
	return t
}

func (i *IconWithMessage) SetIcon(icon fyne.Resource) {
	fyne.Do(func() {
		if icon == nil {
			i.Icon.Hide()
		} else {
			i.Icon.SetResource(icon)
			i.Icon.Show()
		}
	})
}

func (i *IconWithMessage) SetMessage(message string, color color.Color) {
	fyne.Do(func() {
		i.Message.Text = message
		i.Message.Color = color
	})
}

func (i *IconWithMessage) MouseIn(*desktop.MouseEvent) {
	i.Message.Show()
}
func (i *IconWithMessage) MouseOut() {
	i.Message.Hide()
}
func (i *IconWithMessage) MouseMoved(*desktop.MouseEvent) {
}
func (i *IconWithMessage) Cursor() desktop.Cursor {
	return desktop.DefaultCursor
}

func (i *IconWithMessage) CreateRenderer() fyne.WidgetRenderer {
	return &iconWithMessageRenderer{
		i: i,
	}
}

type iconWithMessageRenderer struct {
	i *IconWithMessage
}

func (r *iconWithMessageRenderer) MinSize() fyne.Size {
	return fyne.NewSize(20, 20)
}

func (r *iconWithMessageRenderer) Layout(size fyne.Size) {
	if r.i.Icon.Visible() {
		r.i.Icon.Resize(size)
		r.i.Icon.Move(fyne.NewPos(0, 0))
	}

	messageSize := r.i.Message.MinSize()
	r.i.Message.Resize(messageSize)
	r.i.Message.Move(fyne.NewPos(min(115-messageSize.Width, -(messageSize.Width-size.Width)/2), size.Height+5))
}

func (r *iconWithMessageRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.i.Icon, r.i.Message}
}

func (r *iconWithMessageRenderer) Refresh() {
	canvas.Refresh(r.i)
}

func (r *iconWithMessageRenderer) Destroy() {
}
