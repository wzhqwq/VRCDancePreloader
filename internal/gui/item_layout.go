package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

type dynamicFrameLayout struct {
	thumbnail fyne.CanvasObject
	info      fyne.CanvasObject
	progress  fyne.CanvasObject
	status    fyne.CanvasObject
	errorText fyne.CanvasObject
}

func (d *dynamicFrameLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalHeight := d.info.MinSize().Height + d.status.MinSize().Height + theme.Padding()
	return fyne.NewSize(playItemMinWidth, totalHeight)
}
func (d *dynamicFrameLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	bottomHeight := d.status.MinSize().Height
	centerHeight := size.Height - bottomHeight - theme.Padding()

	rightHeight := centerHeight
	if !d.errorText.Visible() {
		rightHeight = size.Height
	}

	leftWidth := size.Width
	if d.progress.Visible() {
		leftWidth -= d.progress.MinSize().Width + theme.Padding()
		d.progress.Resize(fyne.NewSize(d.progress.MinSize().Width, rightHeight))
		d.progress.Move(fyne.NewPos(leftWidth+theme.Padding(), 0))
	}

	infoWidth := leftWidth
	infoX := float32(0)
	if size.Width > playItemThumbnailShow {
		thumbnailWidth := playItemThumbnailStartWidth + size.Width - playItemThumbnailShow
		if thumbnailWidth > playItemThumbnailMaxWidth {
			thumbnailWidth = playItemThumbnailMaxWidth
		}

		d.thumbnail.Resize(fyne.NewSize(thumbnailWidth, size.Height))

		infoWidth -= thumbnailWidth + theme.Padding()
		infoX = thumbnailWidth + theme.Padding()
	} else {
		d.thumbnail.Resize(fyne.NewSize(0, centerHeight))
	}
	d.info.Resize(fyne.NewSize(infoWidth, centerHeight))
	d.info.Move(fyne.NewPos(infoX, 0))
	d.status.Move(fyne.NewPos(infoX, centerHeight+theme.Padding()))

	if d.errorText.Visible() {
		errorX := infoX + d.status.MinSize().Width + theme.Padding()
		d.errorText.Resize(fyne.NewSize(size.Width-errorX, bottomHeight))
		d.errorText.Move(fyne.NewPos(errorX, centerHeight+theme.Padding()))
	}
}

func NewDynamicFrame(
	thumbnail fyne.CanvasObject,
	info fyne.CanvasObject,
	progress fyne.CanvasObject,
	status fyne.CanvasObject,
	errorText fyne.CanvasObject,
) *fyne.Container {
	layout := &dynamicFrameLayout{
		thumbnail: thumbnail,
		info:      info,
		progress:  progress,
		status:    status,
		errorText: errorText,
	}
	return container.New(layout, thumbnail, info, progress, status, errorText)
}
