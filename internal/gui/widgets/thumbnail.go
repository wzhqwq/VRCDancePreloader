package widgets

import (
	"io"
	"log"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

func GetThumbnailURL(videoURL string) string {
	youtubeID, isYoutube := utils.CheckYoutubeURL(videoURL)
	if isYoutube {
		return utils.GetYoutubeMQThumbnailURL(youtubeID)
	}

	return "https://via.placeholder.com/150"
}
func GetThumbnailImage(videoURL string) fyne.Resource {
	log.Println("Get: ", GetThumbnailURL(videoURL))
	resp, err := http.Get(GetThumbnailURL(videoURL))
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
	VideoURL string
}

type thumbnailRenderer struct {
	image    *canvas.Image
	videoURL string

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
	if r.t.VideoURL != r.videoURL {
		r.LoadImage()
	}
}

func (r *thumbnailRenderer) LoadImage() {
	r.videoURL = r.t.VideoURL
	go func() {
		image := GetThumbnailImage(r.t.VideoURL)
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

func NewThumbnail(videoURL string) *Thumbnail {
	return &Thumbnail{
		VideoURL: videoURL,
	}
}
