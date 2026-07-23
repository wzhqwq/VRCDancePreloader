package widgets

import (
	"image"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/widgets/interactive_widgets"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type Thumbnail struct {
	interactive_widgets.LifeCycleWidget

	id string

	image image.Image

	imageChanged bool

	idArrived chan struct{}
}

func NewThumbnailWithID(id string) *Thumbnail {
	t := &Thumbnail{
		id:    id,
		image: thumbnails.GetGroupThumbnail("default"),

		idArrived: make(chan struct{}),
	}
	t.ExtendBaseWidget(t)
	t.AddLifeCycleFn(t.loop)

	return t
}

func (t *Thumbnail) setImage(image image.Image) {
	t.imageChanged = true
	t.image = image
	fyne.Do(t.Refresh)
}

func (t *Thumbnail) loop(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		default:
			t.loopForId(stopCh)
		}
	}
}

func (t *Thumbnail) loopForId(stopCh <-chan struct{}) {
	provider := third_parties.GetProviderById(t.id)
	if provider == nil {
		return
	}

	thumbnailHandle := provider.Thumbnail(t.id)
	defer thumbnailHandle.Release()
	infoHandle := provider.Info(t.id)
	defer infoHandle.Release()

	var infoChannel chan interactive.RemoteSnapshot[types.GeneralVideoInfo]

	thumbnailSnap := thumbnailHandle.Snapshot()
	if thumbnailSnap.HasData {
		t.setImage(thumbnailSnap.Data)
		return
	}

	infoSnap := infoHandle.Snapshot()
	group := infoSnap.Data.GroupName
	t.setImage(thumbnails.GetGroupThumbnail(group))

	if group == "" || strings.HasPrefix(group, "default") {
		infoCh := infoHandle.Subscribe()
		infoChannel = infoCh.Channel
		defer infoCh.Close()
	}

	thumbnailCh := thumbnailHandle.Subscribe()
	defer thumbnailCh.Close()

	for {
		select {
		case <-stopCh:
			return
		case <-t.idArrived:
			return
		case thumbnailSnap = <-thumbnailCh.Channel:
			if thumbnailSnap.HasData {
				t.setImage(thumbnailSnap.Data)
				return
			}
		case infoSnap = <-infoChannel:
			if infoSnap.Data.GroupName != group {
				group = infoSnap.Data.GroupName
				t.setImage(thumbnails.GetGroupThumbnail(group))
			}
		}
	}
}

func (t *Thumbnail) LoadImageFromID(id string) {
	t.id = id
	select {
	case t.idArrived <- struct{}{}:
	default:
	}
}

func (t *Thumbnail) CreateRenderer() fyne.WidgetRenderer {
	i := canvas.NewImageFromImage(t.image)
	i.FillMode = canvas.ImageFillContain

	r := &thumbnailRenderer{t: t, i: i}
	r.Created(t)

	return r
}

type thumbnailRenderer struct {
	interactive_widgets.BaseLifeCycleRenderer

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
		r.i = canvas.NewImageFromImage(r.t.image)
		r.i.FillMode = canvas.ImageFillContain
		canvas.Refresh(r.t)
	}
}

func (r *thumbnailRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.i}
}
