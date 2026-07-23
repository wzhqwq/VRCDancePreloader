package third_parties

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var ErrFeatureDisabled = fmt.Errorf("%wyou've disabled the feature", interactive.ErrUnrecoverable)
var ErrUnexpectedParam = fmt.Errorf("%wunexpected param", interactive.ErrUnrecoverable)
var ErrRefused = fmt.Errorf("%wthe remote server refuse to provide a video", interactive.ErrTemporarilyUnavailable)
var ErrYtDlpNotAvailable = errors.New("YtDlp not available")

var ErrCatalogUnavailable = errors.New("catalog unavailable")

const ModeDisabled = "disabled"
const ModeApi = "api"
const ModeYtDlp = "ytdlp"

const ResourceVideo = "video"
const ResourceInfo = "info"
const ResourceThumbnail = "thumbnail"
const ResourceCatalog = "catalog"

type ResourceProvider interface {
	SetAllowResources([]string)
	SetMode(string)

	Info(id string) *interactive.RemoteHandle[types.GeneralVideoInfo]
	ModifyInfoPlaceholder(id string, modify func(current types.GeneralVideoInfo) types.GeneralVideoInfo)
	Thumbnail(id string) *interactive.RemoteHandle[image.Image]
	ResolvedVideo(id string) *interactive.RemoteHandle[*types.RemoteHttpResourceInfo]

	Close()
}

type BaseProvider struct {
	allowResources []string

	videoAvailableEm, thumbnailAvailableEm *utils.EventManager[bool]

	allowedResourcesEm *utils.EventManager[[]string]

	infoManager          *interactive.RemoteManager[types.GeneralVideoInfo]
	thumbnailManager     *interactive.RemoteManager[image.Image]
	resolvedVideoManager *interactive.RemoteManager[*types.RemoteHttpResourceInfo]

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func (p *BaseProvider) setup(name string) {
	p.infoManager.BindLogger(utils.NewLogger(name + " Video Info"))
	p.infoManager.BindScheduler(utils.SharedVideoScheduler())

	p.thumbnailManager.BindAvailability(p.thumbnailAvailableEm.SubscribeEvent)
	p.thumbnailManager.BindLogger(utils.NewLogger(name + " Thumbnail"))
	p.thumbnailManager.BindScheduler(utils.SharedThumbnailScheduler())

	p.resolvedVideoManager.BindAvailability(p.videoAvailableEm.SubscribeEvent)
	p.resolvedVideoManager.BindLogger(utils.NewLogger(name + " Resolver"))
	p.resolvedVideoManager.BindScheduler(utils.SharedVideoScheduler())
}

func (p *BaseProvider) SetAllowResources(resources []string) {
	if p.allowResources != nil {
		changed := false
		for _, resource := range resources {
			if !lo.Contains(p.allowResources, resource) {
				changed = true
				break
			}
		}
		if len(p.allowResources) == len(resources) && !changed {
			return
		}
	}

	p.allowResources = resources

	p.allowedResourcesEm.NotifySubscribers(resources)
}

func (p *BaseProvider) Info(id string) *interactive.RemoteHandle[types.GeneralVideoInfo] {
	return p.infoManager.Acquire(id)
}

func (p *BaseProvider) ModifyInfoPlaceholder(id string, modify func(current types.GeneralVideoInfo) types.GeneralVideoInfo) {
	p.infoManager.ModifyPlaceholder(id, modify)
}

func (p *BaseProvider) Thumbnail(id string) *interactive.RemoteHandle[image.Image] {
	return p.thumbnailManager.Acquire(id)
}
func (p *BaseProvider) ResolvedVideo(id string) *interactive.RemoteHandle[*types.RemoteHttpResourceInfo] {
	return p.resolvedVideoManager.Acquire(id)
}

func (p *BaseProvider) Close() {
	close(p.stopCh)
	p.wg.Wait()
}

func getDefaultThumbnail(_ string) image.Image {
	return thumbnails.GetGroupThumbnail("")
}

type resourceGetters interface {
	getInfoPlaceholder(id string) types.GeneralVideoInfo
	getInfo(id string, _ context.Context) (types.GeneralVideoInfo, error)
	getThumbnail(id string, ctx context.Context) (image.Image, error)
	resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error)
}

func constructBaseProvider[P resourceGetters](p P, infoParallel int) BaseProvider {
	return BaseProvider{
		videoAvailableEm:     utils.NewEventManager[bool](),
		thumbnailAvailableEm: utils.NewEventManager[bool](),
		allowedResourcesEm:   utils.NewEventManager[[]string](),

		infoManager:          interactive.NewRemoteManager(p.getInfo, p.getInfoPlaceholder, 100, infoParallel),
		thumbnailManager:     interactive.NewRemoteManager(p.getThumbnail, getDefaultThumbnail, 100, 3),
		resolvedVideoManager: interactive.NewRemoteManager(p.resolveVideoUrl, nil, 100, 1),

		stopCh: make(chan struct{}),
	}
}
