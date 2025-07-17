package containers

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const contentGap = float32(6)

type ResponsiveFrame struct {
	widget.BaseWidget

	Objects []fyne.CanvasObject

	TwoColumnLimit  int
	ContentMinWidth float32
}

func (r *ResponsiveFrame) CreateRenderer() fyne.WidgetRenderer {
	return &ResponsiveFrameRenderer{
		r: r,
	}
}

func NewResponsiveFrame(twoColumnLimit int, contentMinWidth float32, objects ...fyne.CanvasObject) *ResponsiveFrame {
	r := &ResponsiveFrame{
		Objects:         objects,
		TwoColumnLimit:  twoColumnLimit,
		ContentMinWidth: contentMinWidth,
	}
	r.ExtendBaseWidget(r)
	return r
}

type ResponsiveFrameRenderer struct {
	r *ResponsiveFrame

	IsTwoColumn bool
}

func (r *ResponsiveFrameRenderer) Layout(size fyne.Size) {
	objects := r.r.Objects
	if size.Width < r.r.ContentMinWidth*2+contentGap*3 {
		accY := contentGap
		contentWidth := size.Width - contentGap*2
		for _, child := range objects {
			height := child.MinSize().Height
			child.Resize(fyne.NewSize(contentWidth, height))
			child.Move(fyne.NewPos(contentGap, accY))
			accY += height + contentGap
		}
		if r.IsTwoColumn {
			r.IsTwoColumn = false
			r.r.Refresh()
		}
	} else {
		accY := contentGap
		contentWidth := (size.Width - contentGap*3) / 2
		secondX := contentGap*2 + contentWidth

		for _, child := range objects[:r.r.TwoColumnLimit] {
			height := child.MinSize().Height
			child.Resize(fyne.NewSize(contentWidth, height))
			child.Move(fyne.NewPos(contentGap, accY))
			accY += height + contentGap
		}

		accY = contentGap

		for _, child := range objects[r.r.TwoColumnLimit:] {
			height := child.MinSize().Height
			child.Resize(fyne.NewSize(contentWidth, height))
			child.Move(fyne.NewPos(secondX, accY))
			accY += height + contentGap
		}
		if !r.IsTwoColumn {
			r.IsTwoColumn = true
			r.r.Refresh()
		}
	}
}

func (r *ResponsiveFrameRenderer) MinSize() fyne.Size {
	objects := r.r.Objects
	if r.IsTwoColumn {
		accYLeft := contentGap
		for _, child := range objects[:r.r.TwoColumnLimit] {
			accYLeft += child.MinSize().Height + contentGap
		}
		accYRight := contentGap
		for _, child := range objects[r.r.TwoColumnLimit:] {
			accYRight += child.MinSize().Height + contentGap
		}
		return fyne.NewSize(contentGap*2+r.r.ContentMinWidth, max(accYLeft, accYRight))
	} else {
		accY := contentGap
		for _, child := range objects {
			accY += child.MinSize().Height + contentGap
		}
		return fyne.NewSize(contentGap*2+r.r.ContentMinWidth, accY)
	}
}

func (r *ResponsiveFrameRenderer) Objects() []fyne.CanvasObject {
	return r.r.Objects
}

func (r *ResponsiveFrameRenderer) Refresh() {
	canvas.Refresh(r.r)
}

func (r *ResponsiveFrameRenderer) Destroy() {}
