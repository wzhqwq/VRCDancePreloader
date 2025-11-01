package button

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
)

type PaddedIconBtn struct {
	widget.BaseWidget
	fyne.Tappable
	desktop.Hoverable

	padding       float32
	minSquareSize float32

	icon    fyne.Resource
	hovered bool

	OnClick   func()
	OnDestroy func()
}

func NewPaddedIconBtn(icon fyne.Resource) *PaddedIconBtn {
	b := &PaddedIconBtn{}

	b.Extend(icon)

	b.ExtendBaseWidget(b)

	return b
}

func (b *PaddedIconBtn) Extend(icon fyne.Resource) {
	b.padding = theme.Padding()
	b.minSquareSize = 24
	b.icon = icon
}

func (b *PaddedIconBtn) SetIcon(icon fyne.Resource) {
	b.icon = icon
	fyne.Do(func() {
		b.Refresh()
	})
}

func (b *PaddedIconBtn) SetPadding(padding float32) {
	b.padding = padding
	fyne.Do(func() {
		b.Refresh()
	})
}

func (b *PaddedIconBtn) SetMinSquareSize(size float32) {
	b.minSquareSize = size
	fyne.Do(func() {
		b.Refresh()
	})
}

func (b *PaddedIconBtn) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameButton))
	background.CornerRadius = 5

	return &paddedIconBtnRenderer{
		btn:        b,
		Icon:       widget.NewIcon(b.icon),
		Background: background,
	}
}

func (b *PaddedIconBtn) Tapped(_ *fyne.PointEvent) {
	if b.OnClick != nil {
		b.OnClick()
	}
}

func (b *PaddedIconBtn) MouseIn(_ *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
}
func (b *PaddedIconBtn) MouseOut() {
	b.hovered = false
	b.Refresh()
}
func (b *PaddedIconBtn) MouseMoved(_ *desktop.MouseEvent) {
}

type paddedIconBtnRenderer struct {
	btn *PaddedIconBtn

	Icon       *widget.Icon
	Background *canvas.Rectangle
}

func (r *paddedIconBtnRenderer) MinSize() fyne.Size {
	return fyne.NewSquareSize(r.btn.minSquareSize)
}
func (r *paddedIconBtnRenderer) Layout(size fyne.Size) {
	r.Background.Resize(size)
	pad := r.btn.padding
	r.Icon.Resize(fyne.NewSize(size.Width-pad*2, size.Height-pad*2))
	r.Icon.Move(fyne.NewPos(pad, pad))
}
func (r *paddedIconBtnRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.Background, r.Icon}
}
func (r *paddedIconBtnRenderer) Refresh() {
	r.Icon.SetResource(r.btn.icon)

	if r.btn.hovered {
		r.Background.FillColor = theme.Color(custom_fyne.ColorNameButtonHover)
	} else {
		r.Background.FillColor = theme.Color(theme.ColorNameButton)
	}
	r.Background.Refresh()
}
func (r *paddedIconBtnRenderer) Destroy() {
	if r.btn.OnDestroy != nil {
		r.btn.OnDestroy()
	}
}
