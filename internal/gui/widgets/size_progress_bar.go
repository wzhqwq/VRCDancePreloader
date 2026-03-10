package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

const cornerRadius = 4
const barGap = 2

type SizeProgressBar struct {
	widget.BaseWidget

	Bar        *canvas.Rectangle
	Background *canvas.Rectangle
	Text       *canvas.Text

	TotalSize   int64
	CurrentSize int64
}

func NewSizeProgressBar(totalSize int64, currentSize int64) *SizeProgressBar {
	bar := canvas.NewRectangle(theme.Color(custom_fyne.ColorNamePrimaryBackground))
	background := canvas.NewRectangle(theme.Color(custom_fyne.ColorNamePrimaryGrayscale))
	text := canvas.NewText("", theme.Color(theme.ColorNameForegroundOnPrimary))
	text.TextSize = 12

	bar.CornerRadius = cornerRadius
	background.CornerRadius = cornerRadius

	g := &SizeProgressBar{
		Bar:        bar,
		Background: background,
		Text:       text,

		TotalSize:   totalSize,
		CurrentSize: currentSize,
	}

	g.ExtendBaseWidget(g)

	return g
}

func (g *SizeProgressBar) SetCurrentSize(size int64) {
	if size < 0 {
		size = 0
	}
	if size == g.CurrentSize {
		return
	}
	g.CurrentSize = size
	g.Refresh()
}

func (g *SizeProgressBar) SetTotalSize(size int64) {
	if size < 0 {
		size = 0
	}
	if size == g.TotalSize {
		return
	}
	g.TotalSize = size
	g.Refresh()
}

func (g *SizeProgressBar) updateBar() {
	ratio := float32(g.CurrentSize) / float32(g.TotalSize)
	totalSize := g.Background.Size()

	if ratio > 1 {
		ratio = 1
		g.Bar.FillColor = theme.Color(theme.ColorNameError)
	} else {
		g.Bar.FillColor = theme.Color(custom_fyne.ColorNamePrimaryBackground)
	}

	g.Bar.Resize(fyne.NewSize(totalSize.Width*ratio, totalSize.Height))
}

func (g *SizeProgressBar) updateText() {
	label := utils.PrettyByteSize(g.CurrentSize) + " / " + utils.PrettyByteSize(g.TotalSize)
	g.Text.Text = label

	textSize := g.Text.MinSize()
	textX := (g.Background.Size().Width - textSize.Width) / 2
	textY := (g.Background.Size().Height - textSize.Height) / 2
	g.Text.Move(fyne.NewPos(textX, textY))
}

func (g *SizeProgressBar) CreateRenderer() fyne.WidgetRenderer {
	return &SizeProgressBarRenderer{
		g: g,
	}
}

type SizeProgressBarRenderer struct {
	g *SizeProgressBar
}

func (r *SizeProgressBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, 20)
}

func (r *SizeProgressBarRenderer) Layout(size fyne.Size) {
	r.g.Background.Resize(size)
	r.g.Background.Move(fyne.NewPos(0, 0))
	r.g.Bar.Move(fyne.NewPos(0, 0))

	r.g.updateBar()
	r.g.updateText()
}

func (r *SizeProgressBarRenderer) Refresh() {
	r.g.updateText()
	r.g.updateBar()
	canvas.Refresh(r.g)
}

func (r *SizeProgressBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{
		r.g.Background,
		r.g.Bar,
		r.g.Text,
	}
}

func (r *SizeProgressBarRenderer) Destroy() {
}
