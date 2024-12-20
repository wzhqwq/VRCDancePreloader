package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/requesting"
	"io"
	"log"
)

func GetThumbnailImage(url string) fyne.Resource {
	log.Println("Get: ", url)
	resp, err := requesting.RequestThumbnail(url)
	if err != nil {
		log.Println("Failed to get thumbnail:", err)
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fyne.LogError("Unable to read image data", err)
		return nil
	}

	return fyne.NewStaticResource("thumbnail", data)
}

type Thumbnail struct {
	widget.BaseWidget
	ThumbnailURL string
	image        *canvas.Image
}

func NewThumbnail(thumbnailURL string) *Thumbnail {
	image := canvas.NewImageFromResource(theme.MediaVideoIcon())
	image.FillMode = canvas.ImageFillContain

	t := &Thumbnail{
		ThumbnailURL: thumbnailURL,
		image:        image,
	}
	t.ExtendBaseWidget(t)

	go t.LoadImage()

	return t
}

func (t *Thumbnail) CreateRenderer() fyne.WidgetRenderer {
	return &thumbnailRenderer{
		t: t,
	}
}

type thumbnailRenderer struct {
	t *Thumbnail
}

func (r *thumbnailRenderer) MinSize() fyne.Size {
	return fyne.NewSize(60, 40)
}

func (r *thumbnailRenderer) Layout(size fyne.Size) {
	if size.Width < 40 {
		r.t.image.Hide()
	} else {
		r.t.image.Show()
		r.t.image.Resize(size)
	}
}

func (r *thumbnailRenderer) Refresh() {
	r.t.image.Refresh()
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.t.image}
}

func (t *Thumbnail) LoadImage() {
	go func() {
		image := GetThumbnailImage(t.ThumbnailURL)
		if image == nil {
			return
		}
		t.image.Resource = image
		t.image.Refresh()
	}()
}

func (r *thumbnailRenderer) Destroy() {
}
