package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"log"
)

var playItemMinWidth float32 = 260
var playItemThumbnailShow float32 = 300
var playItemThumbnailStartWidth float32 = 60
var playItemThumbnailMaxWidth float32 = 120

type dynamicFrameLayout struct {
	thumbnail fyne.CanvasObject
	info      fyne.CanvasObject
	progress  fyne.CanvasObject
	status    fyne.CanvasObject
	errorText fyne.CanvasObject
}

func (d *dynamicFrameLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	totalHeight := d.info.MinSize().Height
	if d.status != nil {
		totalHeight += d.status.MinSize().Height + theme.Padding()
	}
	return fyne.NewSize(playItemMinWidth, totalHeight)
}
func (d *dynamicFrameLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	bottomHeight := float32(0)
	if d.status != nil {
		bottomHeight = d.status.MinSize().Height
	}
	centerHeight := size.Height - bottomHeight - theme.Padding()

	rightHeight := centerHeight
	if d.errorText == nil || !d.errorText.Visible() {
		rightHeight = size.Height
	}

	leftWidth := size.Width
	if d.progress != nil && d.progress.Visible() {
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
	if d.status != nil {
		d.status.Move(fyne.NewPos(infoX, centerHeight+theme.Padding()))
	}

	if d.errorText != nil && d.errorText.Visible() {
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
	if thumbnail == nil {
		log.Fatal("thumbnail is required")
	}
	if info == nil {
		log.Fatal("info is required")
	}
	layout := &dynamicFrameLayout{
		thumbnail: thumbnail,
		info:      info,
		progress:  progress,
		status:    status,
		errorText: errorText,
	}
	objects := []fyne.CanvasObject{thumbnail, info}
	if progress != nil {
		objects = append(objects, progress)
	}
	if status != nil {
		objects = append(objects, status)
	}
	if errorText != nil {
		objects = append(objects, errorText)
	}
	return container.New(layout, objects...)
}
