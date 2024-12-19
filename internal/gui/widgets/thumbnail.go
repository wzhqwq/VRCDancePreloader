package widgets

import (
	"io"
	"log"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func GetThumbnailImage(url string) fyne.Resource {
	log.Println("Get: ", url)
	resp, err := http.Get(url)
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
}

type thumbnailRenderer struct {
	image        *canvas.Image
	thumbnailURL string

	t *Thumbnail
}

func (r *thumbnailRenderer) MinSize() fyne.Size {
	return fyne.NewSize(60, 40)
}

func (r *thumbnailRenderer) Layout(size fyne.Size) {
	if size.Width < 40 {
		r.image.Hide()
	} else {
		r.image.Show()
		r.image.Resize(size)
	}
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.image}
}

func (r *thumbnailRenderer) Refresh() {
	if r.t.ThumbnailURL != r.thumbnailURL {
		r.LoadImage()
	}
}

func (r *thumbnailRenderer) LoadImage() {
	r.thumbnailURL = r.t.ThumbnailURL
	go func() {
		image := GetThumbnailImage(r.t.ThumbnailURL)
		if image == nil {
			return
		}
		r.image.Resource = image
		r.image.Refresh()
	}()
}

func (r *thumbnailRenderer) Destroy() {
}

func (t *Thumbnail) CreateRenderer() fyne.WidgetRenderer {
	t.ExtendBaseWidget(t)
	image := canvas.NewImageFromResource(theme.MediaVideoIcon())
	image.FillMode = canvas.ImageFillContain
	r := &thumbnailRenderer{
		image: image,
		t:     t,
	}
	r.LoadImage()

	return r
}

func NewThumbnail(thumbnailURL string) *Thumbnail {
	return &Thumbnail{
		ThumbnailURL: thumbnailURL,
	}
}
