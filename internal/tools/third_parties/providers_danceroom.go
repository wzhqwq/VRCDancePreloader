package third_parties

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/catalog"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type RoomProvider[T any] struct {
	BaseProvider

	catalogAvailableEm *utils.EventManager[bool]

	catalogAvailable                   *BothTrue
	videoAvailable, thumbnailAvailable *IsAllowed

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
	p.catalogManager.SetAllowed(lo.Contains(p.allowResources, ResourceCatalog))
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

	ytdlpAvailableCh := local_executables.SubscribeYtDlpAvailability()
	defer ytdlpAvailableCh.Close()

	catalogStatusCh := p.catalogHandle.SubscribeStatus()
	defer catalogStatusCh.Close()

	catalogAllowed := lo.Contains(p.allowResources, ResourceCatalog)
	thumbnailAllowed := lo.Contains(p.allowResources, ResourceThumbnail)
	videoAllowed := lo.Contains(p.allowResources, ResourceVideo)

	p.catalogAvailable = newBothTrue(catalogAllowed, p.catalogHandle.Snapshot().Status.Valid(), p.catalogAvailableEm)
	p.thumbnailAvailable = newIsAllowed(thumbnailAllowed, p.thumbnailAvailableEm)
	p.videoAvailable = newIsAllowed(videoAllowed, p.videoAvailableEm)

	for {
		select {
		case <-p.stopCh:
			return
		case allowed := <-resourcesCh.Channel:
			catalogAllowed = lo.Contains(allowed, ResourceCatalog)
			thumbnailAllowed = lo.Contains(allowed, ResourceThumbnail)
			videoAllowed = lo.Contains(allowed, ResourceVideo)

			if catalogAllowed != p.catalogAvailable.allowed {
				p.catalogManager.SetAllowed(catalogAllowed)
			}
			p.catalogAvailable.SetAllowed(catalogAllowed)
			p.thumbnailAvailable.SetAllowed(thumbnailAllowed)
			p.videoAvailable.SetAllowed(videoAllowed)
		case catalogStatus := <-catalogStatusCh.Channel:
			p.catalogAvailable.SetAvailable(catalogStatus.Valid())
		// client changed
		case <-clientCh.Channel:
			p.catalogAvailable.NotifyIfSatisfied()
			p.thumbnailAvailable.NotifyIfAllowed()
			p.videoAvailable.NotifyIfAllowed()

			if catalogAllowed {
				p.catalogManager.SetAllowed(true)
			}
		}
	}
}

func (p *RoomProvider[T]) findSong(id int) (song T, ok bool) {
	c := p.catalogHandle.Snapshot().Data
	if c == nil {
		return
	}

	song, ok = c.FindSong(id)
	return
}

func (p *RoomProvider[T]) setup(name string, client *requesting.ClientProvider) {
	p.BaseProvider.setup(name)
	p.infoManager.BindAvailability(p.catalogAvailableEm.SubscribeEvent)
	p.client = client
}

func constructRoomProvider[T any, P resourceGetters](p P, catalogManager catalog.Manager[T]) RoomProvider[T] {
	return RoomProvider[T]{
		BaseProvider:       constructBaseProvider(p, 10),
		catalogAvailableEm: utils.NewEventManager[bool](),

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
	if !lo.Contains(p.allowResources, ResourceCatalog) {
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
	if !lo.Contains(p.allowResources, ResourceThumbnail) {
		return nil, errPyPyDanceThumbnailDisabled
	}

	if pypyId, isPypy := internal_id.CheckIdIsPyPy(id); isPypy {
		i, err := thumbnails.GetThumbnailImage(requesting.GetClient(requesting.PyPyDance), internal_id.GetPyPyThumbnailUrl(pypyId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get PyPyDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *PyPyDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.allowResources, ResourceVideo) {
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
	if !lo.Contains(p.allowResources, ResourceCatalog) {
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
	if !lo.Contains(p.allowResources, ResourceThumbnail) {
		return nil, errWannaDanceThumbnailDisabled
	}

	if wannaId, isWanna := internal_id.CheckIdIsWanna(id); isWanna {
		i, err := thumbnails.GetThumbnailImage(requesting.GetClient(requesting.WannaDance), internal_id.GetWannaThumbnailUrl(wannaId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get WannaDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *WannaDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.allowResources, ResourceVideo) {
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
	if !lo.Contains(p.allowResources, ResourceCatalog) {
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
	if !lo.Contains(p.allowResources, ResourceThumbnail) {
		return nil, errDuDuFitDanceThumbnailDisabled
	}

	if duduId, isDuDu := internal_id.CheckIdIsDuDu(id); isDuDu {
		i, err := thumbnails.GetThumbnailImage(requesting.GetClient(requesting.DuDuFitDance), internal_id.GetDuDuThumbnailUrl(duduId), ctx)
		if err != nil {
			return nil, fmt.Errorf("get DuDuFitDance thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *DuDuFitDanceProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if !lo.Contains(p.allowResources, ResourceVideo) {
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
