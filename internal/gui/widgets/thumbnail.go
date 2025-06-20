package widgets

import (
	"bytes"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/nfnt/resize"
	"github.com/stephennancekivell/go-future/future"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/icons"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"image"
	"image/jpeg"
	"io"
	"log"
)

type AsyncImage struct {
	i      future.Future[image.Image]
	loaded bool
}

var cache = utils.NewWeakCache[AsyncImage](100)

func GetThumbnailImage(url string) image.Image {
	if i, ok := cache.Get(url); ok {
		return i.i.Get()
	}

	i := future.New(func() image.Image {
		defer func() {
			if i, ok := cache.Get(url); ok {
				i.loaded = true
			}
		}()

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

		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			log.Println("Failed to decode image:", err)
			return nil
		}

		return resize.Resize(320, 0, img, resize.Bilinear)
	})

	cache.Set(url, AsyncImage{i: i, loaded: false})

	return i.Get()
}

func HasThumbnailCachedAndLoaded(url string) bool {
	f, ok := cache.Get(url)
	if ok {
		return f.loaded
	}
	return false
}

type Thumbnail struct {
	widget.BaseWidget

	thumbnailURL string
	ID           string

	loading bool

	image image.Image

	imageChanged bool
}

func NewThumbnail(thumbnailURL string) *Thumbnail {
	t := &Thumbnail{
		thumbnailURL: thumbnailURL,
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
	if r.t.loading || r.t.image == nil {
		r.i.Resize(fyne.NewSize(40, 40))
		r.i.Move(fyne.NewPos(size.Width/2-20, size.Height/2-20))
	} else {
		r.i.Resize(size)
		r.i.Move(fyne.NewPos(0, 0))
	}
}

func (r *thumbnailRenderer) Refresh() {
	if r.t.imageChanged {
		r.t.imageChanged = false
		if r.t.loading {
			// TODO spinner
			r.i = canvas.NewImageFromResource(icons.GetIcon("movie"))
		} else if r.t.image == nil {
			r.i = canvas.NewImageFromResource(icons.GetIcon("movie"))
		} else {
			r.i = canvas.NewImageFromImage(r.t.image)
			r.i.FillMode = canvas.ImageFillContain
		}
		canvas.Refresh(r.t)
	}
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.i}
}

func (t *Thumbnail) loadImage() {
	t.imageChanged = true

	if t.thumbnailURL == "" {
		if t.ID == "" {
			t.loading = false
		} else {
			t.loading = true
			defer func() {
				t.thumbnailURL = third_party_api.GetThumbnailByInternalID(t.ID).Get()
				go t.loadImage()
			}()
		}
		t.image = nil
	} else if HasThumbnailCachedAndLoaded(t.thumbnailURL) {
		t.loading = false
		t.image = GetThumbnailImage(t.thumbnailURL)
	} else {
		t.loading = true
		t.image = nil

		defer func() {
			if t.thumbnailURL == "" {
				return
			}

			i := GetThumbnailImage(t.thumbnailURL)
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
	t.thumbnailURL = url
	go t.loadImage()
}
func (t *Thumbnail) LoadImageFromID(id string) {
	t.ID = id
	t.thumbnailURL = ""
	go t.loadImage()
}

func (r *thumbnailRenderer) Destroy() {
}
