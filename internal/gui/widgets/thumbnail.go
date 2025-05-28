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

		return resize.Resize(160, 0, img, resize.Bilinear)
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
	showIcon     bool
	i            *canvas.Image
}

func NewThumbnail(thumbnailURL string) *Thumbnail {
	t := &Thumbnail{
		thumbnailURL: thumbnailURL,
	}
	t.ExtendBaseWidget(t)

	t.LoadImage()

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
	r.t.i.Show()
	if r.t.showIcon {
		r.t.i.Resize(fyne.NewSize(40, 40))
		r.t.i.Move(fyne.NewPos(size.Width/2-20, size.Height/2-20))
	} else {
		r.t.i.Resize(size)
		r.t.i.Move(fyne.NewPos(0, 0))
	}
}

func (r *thumbnailRenderer) Refresh() {
	canvas.Refresh(r.t)
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.t.i}
}

func (t *Thumbnail) LoadImage() {
	if t.thumbnailURL == "" {
		t.showIcon = true
		t.i = canvas.NewImageFromResource(icons.GetIcon("movie"))
		return
	}
	if HasThumbnailCachedAndLoaded(t.thumbnailURL) {
		t.showIcon = false
		t.i = canvas.NewImageFromImage(GetThumbnailImage(t.thumbnailURL))
		t.i.FillMode = canvas.ImageFillContain
	} else {
		t.showIcon = true
		t.i = canvas.NewImageFromResource(icons.GetIcon("movie"))

		go func() {
			if t.thumbnailURL == "" {
				return
			}

			i := GetThumbnailImage(t.thumbnailURL)
			if i == nil {
				return
			}

			t.i = canvas.NewImageFromImage(i)
			t.i.FillMode = canvas.ImageFillContain
			t.showIcon = false

			fyne.Do(func() {
				t.Refresh()
			})
		}()
	}
}

func (t *Thumbnail) LoadImageFromURL(url string) {
	t.thumbnailURL = url
	t.LoadImage()
}

func (r *thumbnailRenderer) Destroy() {
}
