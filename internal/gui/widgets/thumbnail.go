package widgets

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
)

type Thumbnail struct {
	widget.BaseWidget

	url string
	ID  string

	loading bool
	invalid bool

	image image.Image

	imageChanged bool
}

func NewThumbnail(thumbnailURL string) *Thumbnail {
	t := &Thumbnail{
		url: thumbnailURL,
	}
	t.ExtendBaseWidget(t)

	return t
}

func NewThumbnailWithID(id string) *Thumbnail {
	t := &Thumbnail{
		ID: id,
	}
	t.ExtendBaseWidget(t)

	return t
}

func (t *Thumbnail) CreateRenderer() fyne.WidgetRenderer {
	go t.loadImage()

	return &thumbnailRenderer{
		t: t,
		i: canvas.NewImageFromResource(icons.GetIcon("movie")),
	}
}

type thumbnailRenderer struct {
	t *Thumbnail

	i *canvas.Image
}

func (r *thumbnailRenderer) MinSize() fyne.Size {
	return fyne.NewSize(60, 40)
}

func (r *thumbnailRenderer) Layout(size fyne.Size) {
	r.i.Resize(size)
	r.i.Move(fyne.NewPos(0, 0))
}

func (r *thumbnailRenderer) Refresh() {
	if r.t.imageChanged {
		r.t.imageChanged = false
		if r.t.loading || r.t.image == nil {
			local := thumbnails.GetThumbnailImage("", third_party_api.GetLocalThumbnailByInternalID(r.t.ID))
			r.i = canvas.NewImageFromImage(local)
		} else {
			r.i = canvas.NewImageFromImage(r.t.image)
		}
		r.i.FillMode = canvas.ImageFillContain
		canvas.Refresh(r.t)
	}
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.i}
}

func (t *Thumbnail) loadImage() {
	t.imageChanged = true

	if t.url == "" {
		if t.ID == "" || t.invalid {
			t.loading = false
		} else if thumbnails.HasThumbnailCachedAndLoaded(t.ID) {
			t.loading = false
			t.image = thumbnails.GetThumbnailImage(t.ID, "")
		} else {
			t.loading = true
			defer func() {
				t.url = third_party_api.GetThumbnailByInternalID(t.ID).Get()
				if t.url == "" {
					t.invalid = true
				}
				go t.loadImage()
			}()
		}
		t.image = nil
	} else if thumbnails.HasThumbnailCachedAndLoaded(t.url) {
		t.loading = false
		t.image = thumbnails.GetThumbnailImage(t.ID, t.url)
	} else {
		t.loading = true

		defer func() {
			if t.url == "" {
				return
			}

			i := thumbnails.GetThumbnailImage(t.ID, t.url)
			if i == nil {
				return
			}

			t.image = i
			t.imageChanged = true
			t.loading = false

			fyne.Do(func() {
				t.Refresh()
			})
		}()
	}

	fyne.DoAndWait(func() {
		t.Refresh()
	})
}

func (t *Thumbnail) LoadImageFromURL(url string) {
	t.url = url
	t.invalid = false
	go t.loadImage()
}
func (t *Thumbnail) LoadImageFromID(id string) {
	t.ID = id
	t.url = ""
	t.invalid = false
	go t.loadImage()
}

func (r *thumbnailRenderer) Destroy() {
}
