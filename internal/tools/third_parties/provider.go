package third_parties

import (
	"context"
	"errors"
	"fmt"
	"image"
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

var ErrFeatureDisabled = fmt.Errorf("%wyou've disabled the feature", interactive.ErrUnrecoverableDisabled)
var ErrUnexpectedParam = fmt.Errorf("%wunexpected param", interactive.ErrUnrecoverable)
var ErrYtDlpNotAvailable = errors.New("YtDlp not available")

var ErrCatalogUnavailable = errors.New("catalog unavailable")

const ModeDisabled = "disabled"
const ModeApi = "api"
const ModeYtDlp = "ytdlp"

const ResourceVideo = "video"
const ResourceInfo = "info"
const ResourceThumbnail = "thumbnail"
const ResourceCatalog = "catalog"

var thumbnailRetryPolicy = utils.RetryPolicy{
	MaxRetries: 3,
	Delay:      time.Second * 3,
	Jitter:     true,
}

type ResourceProvider interface {
	SetAllowResources([]string)
	SetMode(string)

	Info(id string) *interactive.RemoteHandle[types.GeneralVideoInfo]
	ModifyInfoPlaceholder(id string, modify func(current types.GeneralVideoInfo) types.GeneralVideoInfo)
	Thumbnail(id string) *interactive.RemoteHandle[image.Image]
	ResolvedVideo(id string) *interactive.RemoteHandle[*types.RemoteHttpResourceInfo]

	Start()
	Close()
}

type BaseProvider struct {
	// allowResources is written by SetAllowResources on the configuration thread
	// and read by every provider loop and every getter (which run on the workers
	// of the managers below). It is a pointer so that both sides can use it
	// without a lock, and so that "nothing allowed yet" is representable: the zero
	// value reads as nil, which allows nothing.
	allowResources atomic.Pointer[[]string]

	allowedResourcesEm *utils.EventManager[[]string]

	// Every provider offers the same three resources, so the gates live here and
	// a provider only supplies their "available" input from its own loop (its
	// "allowed" input is always the resource list).
	infoAvailable      *ResourceGate
	thumbnailAvailable *ResourceGate
	videoAvailable     *ResourceGate

	infoManager          *interactive.RemoteManager[types.GeneralVideoInfo]
	thumbnailManager     *interactive.RemoteManager[image.Image]
	resolvedVideoManager *interactive.RemoteManager[*types.RemoteHttpResourceInfo]

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func (p *BaseProvider) setup(name string) {
	p.infoAvailable = newResourceGate()
	p.thumbnailAvailable = newResourceGate()
	p.videoAvailable = newResourceGate()

	p.infoManager.BindLogger(utils.NewLogger(name + " Video Info"))
	p.infoManager.BindAvailability(p.infoAvailable)

	p.thumbnailManager.BindAvailability(p.thumbnailAvailable)
	p.thumbnailManager.BindLogger(utils.NewLogger(name + " Thumbnail"))
	p.thumbnailManager.BindScheduler(utils.SharedThumbnailScheduler())
	p.thumbnailManager.BindRetry(thumbnailRetryPolicy)

	p.resolvedVideoManager.BindAvailability(p.videoAvailable)
	p.resolvedVideoManager.BindLogger(utils.NewLogger(name + " Resolver"))
	p.resolvedVideoManager.BindScheduler(utils.SharedVideoScheduler())
}

// updateGate recomputes one gate. The permission always comes from the resource
// list, so it is derived here rather than passed in: that pair of booleans is
// what got swapped in the previous version (see review/09 §6).
func (p *BaseProvider) updateGate(gate *ResourceGate, resources []string, key string, available ResourceAvailable) {
	gate.Update(ResourceAllowed(lo.Contains(resources, key)), available)
}

// retryGates asks the downstream managers to re-check their entries even though
// the gates themselves did not change.
func (p *BaseProvider) retryGates() {
	p.infoAvailable.Retry()
	p.thumbnailAvailable.Retry()
	p.videoAvailable.Retry()
}

func (p *BaseProvider) SetAllowResources(resources []string) {
	if !utils.IsArrayChanged(p.currentAllowResources(), resources) {
		return
	}

	// Stored before the notification: the loops and the getters re-read the
	// allowed set when they react to it, so the set they then find has to be the
	// one that was announced.
	p.allowResources.Store(&resources)
	p.allowedResourcesEm.NotifySubscribers(resources)
}

// currentAllowResources is the allowed resource set every reader sees. It is safe
// to call from any goroutine; see the note on the field.
func (p *BaseProvider) currentAllowResources() []string {
	if resources := p.allowResources.Load(); resources != nil {
		return *resources
	}

	return nil
}

func (p *BaseProvider) Info(id string) *interactive.RemoteHandle[types.GeneralVideoInfo] {
	return p.infoManager.Acquire(id)
}

func (p *BaseProvider) ModifyInfoPlaceholder(id string, modify func(current types.GeneralVideoInfo) types.GeneralVideoInfo) {
	p.infoManager.ModifyPlaceholderFn(id, modify)
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
	p.infoManager.Close()
	p.thumbnailManager.Close()
	p.resolvedVideoManager.Close()
}

type resourceGetters interface {
	getInfoPlaceholder(id string) types.GeneralVideoInfo
	getInfo(id string, _ context.Context) (types.GeneralVideoInfo, error)
	getThumbnail(id string, ctx context.Context) (image.Image, error)
	resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error)
}

func constructBaseProvider[P resourceGetters](p P, infoParallel int) BaseProvider {
	return BaseProvider{
		allowedResourcesEm: utils.NewEventManager[[]string](),

		infoManager:          interactive.NewRemoteManager(p.getInfo, p.getInfoPlaceholder, 100, infoParallel),
		thumbnailManager:     interactive.NewRemoteManager(p.getThumbnail, nil, 100, 3),
		resolvedVideoManager: interactive.NewRemoteManager(p.resolveVideoUrl, nil, 100, 1),

		stopCh: make(chan struct{}),
	}
}
