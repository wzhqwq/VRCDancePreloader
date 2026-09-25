package third_parties

import (
	"context"
	"fmt"
	"image"
	"strings"
	"sync/atomic"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/catalog"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type RoomProvider[T any] struct {
	BaseProvider

	catalogManager catalog.Manager[T]
	catalogHandle  *interactive.RemoteHandle[*catalog.Catalog[T]]

	client *requesting.ClientProvider
}

func ValidRoomResources(resources []string) bool {
	for _, resource := range resources {
		if resource != ResourceVideo && resource != ResourceCatalog && resource != ResourceThumbnail {
			return false
		}
	}
	return true
}

func (p *RoomProvider[T]) SetMode(_ string) {
}

func (p *RoomProvider[T]) Start() {
	p.catalogManager.SetAllowed(lo.Contains(p.currentAllowResources(), ResourceCatalog))
	p.catalogHandle = p.catalogManager.Handle()

	p.wg.Go(p.loop)
}

func (p *RoomProvider[T]) Close() {
	p.BaseProvider.Close()
	p.catalogHandle.Release()
}

func (p *RoomProvider[T]) loop() {
	resourcesCh := p.allowedResourcesEm.SubscribeEvent()
	defer resourcesCh.Close()

	clientCh := p.client.SubscribeChange()
	defer clientCh.Close()

	catalogCh := p.catalogHandle.Subscribe()
	defer catalogCh.Close()

	p.refreshGates()

	for {
		select {
		case <-p.stopCh:
			return
		// The resource list changed: every gate's permission moved, and the
		// catalog manager keeps its own "may we download the catalog at all".
		case <-resourcesCh.Channel:
			p.catalogManager.SetAllowed(lo.Contains(p.currentAllowResources(), ResourceCatalog))
			p.refreshGates()
		// The catalog entry moved. Its payload is deliberately unused: the gate
		// is recomputed from the entry's current snapshot, so a dropped or
		// outdated event cannot leave a stale value behind.
		case <-catalogCh.Channel:
			p.refreshGates()
		// A new HTTP client replaced the previous one: the fetches that failed
		// with the old one are worth another try. The catalog entry has its own
		// gate, so it has to be told separately.
		case <-clientCh.Channel:
			p.retryGates()

			if lo.Contains(p.currentAllowResources(), ResourceCatalog) {
				p.catalogManager.SetAllowed(true)
			}
		}
	}
}

// refreshGates recomputes all three gates from the current state. Nothing is
// cached here on purpose: the previous version kept copies of the inputs that
// were sampled once when the loop started, and the stale copy is what made a
// finished catalog fetch unable to satisfy the info gate (review/09 §6).
func (p *RoomProvider[T]) refreshGates() {
	resources := p.currentAllowResources()
	catalogReady := ResourceAvailable(p.catalogHandle.Snapshot().HasData)

	p.updateGate(p.infoAvailable, resources, ResourceCatalog, catalogReady)
	// Thumbnails and videos are fetched straight from the room's CDN, so the
	// only thing that can hold them back is the user's own resource list.
	p.updateGate(p.thumbnailAvailable, resources, ResourceThumbnail, true)
	p.updateGate(p.videoAvailable, resources, ResourceVideo, true)
}

func (p *RoomProvider[T]) findSong(id int) (song T, ok bool) {
	snap := p.catalogHandle.Snapshot()
	if !snap.HasData || snap.Data == nil {
		return
	}

	song, ok = snap.Data.FindSong(id)
	return
}

func (p *RoomProvider[T]) setup(name string, client *requesting.ClientProvider) {
	p.BaseProvider.setup(name)
	p.client = client
}

func constructRoomProvider[T any, P resourceGetters](p P, catalogManager catalog.Manager[T]) RoomProvider[T] {
	return RoomProvider[T]{
		BaseProvider: constructBaseProvider(p, 10),

		catalogManager: catalogManager,
	}
}

// PyPyDance

var (
	errPyPyDanceCatalogDisabled   = fmt.Errorf("%w: fetching PyPyDance catalog", ErrFeatureDisabled)
	errPyPyDanceThumbnailDisabled = fmt.Errorf("%w: fetching PyPyDance thumbnail", ErrFeatureDisabled)
	errPyPyDanceVideoDisabled     = fmt.Errorf("%w: fetching PyPyDance video", ErrFeatureDisabled)
)

type PyPyDanceProvider struct {
	RoomProvider[raw_song.PyPyDanceSong]
}

func (*PyPyDanceProvider) getInfoPlaceholder(id string) types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title: "PyPyDance " + strings.TrimPrefix(id, internal_id.PyPyInternalPrefix),
	}
}

func (p *PyPyDanceProvider) getInfo(id string, _ context.Context) (types.GeneralVideoInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceCatalog) {
		return types.GeneralVideoInfo{}, errPyPyDanceCatalogDisabled
	}

	if pypyId, isPypy := internal_id.CheckIdIsPyPy(id); isPypy {
		song, ok := p.findSong(pypyId)
		if !ok {
			return types.GeneralVideoInfo{}, ErrCatalogUnavailable
		}
		fallback := ""
		if len(song.OriginalURL) > 0 {
			fallback = song.OriginalURL[0]
		}
		return types.GeneralVideoInfo{
			Title:     song.Name,
			Duration:  time.Duration(song.End) * time.Second,
			GroupName: song.GroupName,

			FallbackUrl: fallback,
		}, nil
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *PyPyDanceProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceThumbnail) {
		return nil, errPyPyDanceThumbnailDisabled
	}

	if pypyId, isPypy := internal_id.CheckIdIsPyPy(id); isPypy {
		i, err := GetThumbnailImage(requesting.GetClient(requesting.PyPyDance), internal_id.GetPyPyThumbnailUrl(pypyId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get PyPyDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *PyPyDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceVideo) {
		return nil, errPyPyDanceVideoDisabled
	}

	if pypyId, isPypy := internal_id.CheckIdIsPyPy(id); isPypy {
		return directResolve(internal_id.GetPyPyVideoUrl(pypyId), requesting.GetClient(requesting.PyPyDance), ctx)
	}

	return nil, ErrUnexpectedParam
}

func newPyPyDanceProvider() ResourceProvider {
	p := &PyPyDanceProvider{}
	p.RoomProvider = constructRoomProvider(p, catalog.GetPyPyDanceCatalogManager())
	p.setup("PyPyDance", requesting.GetClient(requesting.PyPyDance))
	p.resolvedVideoManager.BindScheduler(utils.PyPyVideoScheduler())

	return p
}

// WannaDance

var (
	errWannaDanceCatalogDisabled   = fmt.Errorf("%w: fetching WannaDance catalog", ErrFeatureDisabled)
	errWannaDanceThumbnailDisabled = fmt.Errorf("%w: fetching WannaDance thumbnail", ErrFeatureDisabled)
	errWannaDanceVideoDisabled     = fmt.Errorf("%w: fetching WannaDance video", ErrFeatureDisabled)
)

type WannaDanceProvider struct {
	RoomProvider[raw_song.WannaDanceSong]
}

func (*WannaDanceProvider) getInfoPlaceholder(id string) types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title: "WannaDance " + strings.TrimPrefix(id, internal_id.WannaInternalPrefix),
	}
}

func (p *WannaDanceProvider) getInfo(id string, _ context.Context) (types.GeneralVideoInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceCatalog) {
		return types.GeneralVideoInfo{}, errWannaDanceCatalogDisabled
	}

	if wannaId, isWanna := internal_id.CheckIdIsWanna(id); isWanna {
		song, ok := p.findSong(wannaId)
		if !ok {
			return types.GeneralVideoInfo{}, ErrCatalogUnavailable
		}
		return types.GeneralVideoInfo{
			Title:     song.FullTitle(),
			Duration:  time.Duration(song.End) * time.Second,
			GroupName: song.Group,
		}, nil
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *WannaDanceProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceThumbnail) {
		return nil, errWannaDanceThumbnailDisabled
	}

	if wannaId, isWanna := internal_id.CheckIdIsWanna(id); isWanna {
		i, err := GetThumbnailImage(requesting.GetClient(requesting.WannaDance), internal_id.GetWannaThumbnailUrl(wannaId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get WannaDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *WannaDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceVideo) {
		return nil, errWannaDanceVideoDisabled
	}

	if wannaId, isWanna := internal_id.CheckIdIsWanna(id); isWanna {
		return directResolve(internal_id.GetWannaVideoUrl(wannaId), requesting.GetClient(requesting.WannaDance), ctx)
	}

	return nil, ErrUnexpectedParam
}

func newWannaDanceProvider() ResourceProvider {
	p := &WannaDanceProvider{}
	p.RoomProvider = constructRoomProvider(p, catalog.GetWannaDanceCatalogManager())
	p.setup("WannaDance", requesting.GetClient(requesting.WannaDance))

	return p
}

// DuDuFitDance

var (
	errDuDuFitDanceCatalogDisabled   = fmt.Errorf("%w: fetching DuDuFitDance catalog", ErrFeatureDisabled)
	errDuDuFitDanceThumbnailDisabled = fmt.Errorf("%w: fetching DuDuFitDance thumbnail", ErrFeatureDisabled)
	errDuDuFitDanceVideoDisabled     = fmt.Errorf("%w: fetching DuDuFitDance video", ErrFeatureDisabled)

	errDuDuFitDanceOriginalVideoDisabled = fmt.Errorf("%w: fetching DuDuFitDance original video info", ErrFeatureDisabled)
)

type DuDuFitDanceProvider struct {
	RoomProvider[raw_song.DuDuFitDanceSong]
}

func (*DuDuFitDanceProvider) getInfoPlaceholder(id string) types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title: "DuDuFitDance " + strings.TrimPrefix(id, internal_id.DuDuInternalPrefix),
	}
}

func (p *DuDuFitDanceProvider) getInfo(id string, _ context.Context) (types.GeneralVideoInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceCatalog) {
		return types.GeneralVideoInfo{}, errDuDuFitDanceCatalogDisabled
	}

	if duduId, isDuDu := internal_id.CheckIdIsDuDu(id); isDuDu {
		song, ok := p.findSong(duduId)
		if !ok {
			return types.GeneralVideoInfo{}, ErrCatalogUnavailable
		}
		return types.GeneralVideoInfo{
			Title:     song.FullTitle(),
			Duration:  time.Duration(song.End) * time.Second,
			GroupName: song.Group,
		}, nil
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *DuDuFitDanceProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceThumbnail) {
		return nil, errDuDuFitDanceThumbnailDisabled
	}

	if duduId, isDuDu := internal_id.CheckIdIsDuDu(id); isDuDu {
		i, err := GetThumbnailImage(requesting.GetClient(requesting.DuDuFitDance), internal_id.GetDuDuThumbnailUrl(duduId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get DuDuFitDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *DuDuFitDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.currentAllowResources(), ResourceVideo) {
		return nil, errDuDuFitDanceVideoDisabled
	}

	if duduId, isDuDu := internal_id.CheckIdIsDuDu(id); isDuDu {
		return directResolve(internal_id.GetDuDuVideoUrl(duduId), requesting.GetClient(requesting.DuDuFitDance), ctx)
	}

	return nil, ErrUnexpectedParam
}

func newDuDuFitDanceProvider() ResourceProvider {
	p := &DuDuFitDanceProvider{}
	p.RoomProvider = constructRoomProvider(p, catalog.GetDuDuFitDanceCatalogManager())
	p.setup("DuDuFitDance", requesting.GetClient(requesting.DuDuFitDance))

	return p
}

type DDFDOriginalVideoInfoProvider struct {
	// enabled is the level its manager reads, and wake tells it to read it again
	// (see interactive.AvailabilitySource). SetEnabled runs on the preloader's
	// configuration goroutine while getInfo runs on the manager's worker, hence
	// the atomic.
	enabled atomic.Bool
	wake    chan struct{}

	manager *interactive.RemoteManager[api.DuDuOriginalVideoInfo]
}

func (p *DDFDOriginalVideoInfoProvider) getInfo(id string, ctx context.Context) (api.DuDuOriginalVideoInfo, error) {
	if !p.enabled.Load() {
		return api.DuDuOriginalVideoInfo{}, errDuDuFitDanceOriginalVideoDisabled
	}

	if duduId, isDuDu := internal_id.CheckIdIsDuDu(id); isDuDu {
		return api.GetDuDuOriginalVideoInfo(duduId, ctx)
	}

	return api.DuDuOriginalVideoInfo{}, ErrUnexpectedParam
}

func (p *DDFDOriginalVideoInfoProvider) Info(id string) *interactive.RemoteHandle[api.DuDuOriginalVideoInfo] {
	return p.manager.Acquire(id)
}

// Available and Wakes make the provider the availability source of its own
// manager: its info may only be fetched while the preloader has it enabled.
func (p *DDFDOriginalVideoInfoProvider) Available() bool {
	return p.enabled.Load()
}

func (p *DDFDOriginalVideoInfoProvider) Wakes() <-chan struct{} {
	return p.wake
}

func (p *DDFDOriginalVideoInfoProvider) SetEnabled(enabled bool) {
	p.enabled.Store(enabled)

	if enabled {
		select {
		case p.wake <- struct{}{}:
		default:
			// A wake-up is already pending and the reader re-reads the level.
		}
	}
}

func (p *DDFDOriginalVideoInfoProvider) Close() {
	p.manager.Close()
}

func NewDDFOriginalVideoInfoProvider() *DDFDOriginalVideoInfoProvider {
	p := &DDFDOriginalVideoInfoProvider{
		wake: make(chan struct{}, 1),
	}
	p.manager = interactive.NewRemoteManager(p.getInfo, nil, 100, 1)
	p.manager.BindAvailability(p)
	p.manager.BindScheduler(utils.DuDuAssetScheduler())
	p.manager.BindLogger(utils.NewLogger("DuDuFitDance Original Video Info"))

	return p
}
