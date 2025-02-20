package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PaddedIconBtn struct {
	widget.BaseWidget
	fyne.Tappable
	desktop.Hoverable

	Icon       *widget.Icon
	Background *canvas.Rectangle

	padding float32

	OnClick func()
}

func NewPaddedIconBtn(icon fyne.Resource) *PaddedIconBtn {
	b := &PaddedIconBtn{}

	b.Extend(icon)

	b.ExtendBaseWidget(b)

	return b
}

func (b *PaddedIconBtn) Extend(icon fyne.Resource) {
	b.Icon = widget.NewIcon(icon)
	b.Background = canvas.NewRectangle(theme.Color(theme.ColorNameButton))
	b.Background.CornerRadius = 5
	b.padding = theme.Padding()
}

func (b *PaddedIconBtn) SetIcon(icon fyne.Resource) {
	b.Icon.SetResource(icon)
}

func (b *PaddedIconBtn) SetPadding(padding float32) {
	b.padding = padding
	b.Refresh()
}

func (b *PaddedIconBtn) CreateRenderer() fyne.WidgetRenderer {
	return &paddedIconBtnRenderer{
		btn: b,
	}
}

func (b *PaddedIconBtn) Tapped(_ *fyne.PointEvent) {
	if b.OnClick != nil {
		b.OnClick()
	}
}

func (b *PaddedIconBtn) MouseIn(_ *desktop.MouseEvent) {
	b.Background.FillColor = theme.Color(theme.ColorNameHover)
	b.Refresh()
}
func (b *PaddedIconBtn) MouseOut() {
	b.Background.FillColor = theme.Color(theme.ColorNameButton)
	b.Refresh()
}
func (b *PaddedIconBtn) MouseMoved(_ *desktop.MouseEvent) {
}

type paddedIconBtnRenderer struct {
	btn *PaddedIconBtn
}

func (r *paddedIconBtnRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(24)
}
func (r *paddedIconBtnRenderer) Layout(size fyne.Size) {
	r.btn.Background.Resize(size)
	pad := r.btn.padding
	r.btn.Icon.Resize(fyne.NewSize(size.Width-pad*2, size.Height-pad*2))
	r.btn.Icon.Move(fyne.NewPos(pad, pad))
}
func (r *paddedIconBtnRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.btn.Background, r.btn.Icon}
}
func (r *paddedIconBtnRenderer) Refresh() {
	r.Layout(r.btn.Size())
}
func (r *paddedIconBtnRenderer) Destroy() {
}
