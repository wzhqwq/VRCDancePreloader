package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"io"
	"log"
	"sync"
	"time"
)

type cachedResource struct {
	res  fyne.Resource
	time time.Time
}

var imageMap = make(map[string]*cachedResource)
var imageMapMutex = sync.Mutex{}
var gcChan = make(chan struct{}, 1)

func GetThumbnailImage(url string) fyne.Resource {
	imageMapMutex.Lock()
	defer imageMapMutex.Unlock()

	if cache, ok := imageMap[url]; ok {
		return cache.res
	}
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

	image := fyne.NewStaticResource("thumbnail", data)
	imageMap[url] = &cachedResource{
		res:  image,
		time: time.Now(),
	}
	//log.Println("Added to image cache: ", url)

	select {
	case gcChan <- struct{}{}:
		go func() {
			imageMapMutex.Lock()
			defer imageMapMutex.Unlock()

			keys := make([]string, 0, len(imageMap))
			for k := range imageMap {
				keys = append(keys, k)
			}
			remaining := len(keys)
			for _, k := range keys {
				if time.Since(imageMap[k].time) > time.Minute*10 {
					delete(imageMap, k)
					//log.Println("Removed from image cache due to timeout: ", k)
				}
				remaining--
			}
			if remaining > 20 {
				keys = make([]string, 0, len(imageMap))
				for k := range imageMap {
					keys = append(keys, k)
				}
				keys = keys[remaining:]
				for _, k := range keys {
					delete(imageMap, k)
					//log.Println("Removed from image cache due to size limit: ", k)
				}
			}
			<-gcChan
		}()
	default:
	}

	return image
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
	if size.Width < 40 {
		r.t.image.Hide()
	} else {
		r.t.image.Show()
		r.t.image.Resize(size)
	}
}

func (r *thumbnailRenderer) Refresh() {
	r.t.LoadImage()
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
