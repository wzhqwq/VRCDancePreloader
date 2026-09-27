package third_parties

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"
	"sync/atomic"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/secrets"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/third_parties/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type PlatformProvider struct {
	BaseProvider

	// mode is read from the request path (the getters below, which run on the
	// info/resolver workers) and from this provider's own loop, while SetMode
	// writes it from the configuration thread. It is a pointer so that both sides
	// can use it without a lock, and so that "never set" is representable: the
	// zero value reads as "".
	mode atomic.Pointer[string]

	modeEm *utils.EventManager[string]
}

// currentMode is the mode every reader sees. It is safe to call from any
// goroutine; see the note on the field.
func (p *PlatformProvider) currentMode() string {
	if mode := p.mode.Load(); mode != nil {
		return *mode
	}

	return ""
}

func ValidMode(mode string) bool {
	return mode == ModeDisabled || mode == ModeApi || mode == ModeYtDlp
}

func (p *PlatformProvider) SetMode(mode string) {
	if mode == p.currentMode() {
		return
	}

	// Stored before the notification: the loop re-reads the mode when it handles
	// the event, so the value it then finds has to be the one that was announced.
	p.mode.Store(&mode)
	p.modeEm.NotifySubscribers(mode)
}

func (p *PlatformProvider) setup(name string) {
	p.BaseProvider.setup(name)
	p.infoManager.BindScheduler(utils.SharedVideoScheduler())
}

func ValidPlatformResources(resources []string) bool {
	for _, resource := range resources {
		if resource != ResourceVideo && resource != ResourceInfo && resource != ResourceThumbnail {
			return false
		}
	}
	return true
}

func constructPlatformProvider[P resourceGetters](p P) PlatformProvider {
	return PlatformProvider{
		modeEm:       utils.NewEventManager[string](),
		BaseProvider: constructBaseProvider(p, 1),
	}
}

// YouTube

var (
	errYouTubeDisabled          = fmt.Errorf("%w: fetching YouTube", ErrFeatureDisabled)
	errYouTubeInfoDisabled      = fmt.Errorf("%w: fetching YouTube info", ErrFeatureDisabled)
	errYouTubeThumbnailDisabled = fmt.Errorf("%w: fetching YouTube thumbnail", ErrFeatureDisabled)
	errYouTubeVideoDisabled     = fmt.Errorf("%w: fetching YouTube video", ErrFeatureDisabled)

	errYouTubeApiNotConfigured = errors.New("YouTube fetching mode set to api, but no api key configured")
	errYouTubeApiNotSupported  = fmt.Errorf("%w: fetching YouTube video through yt-dlp", ErrFeatureDisabled)
)

type YouTubeProvider struct {
	PlatformProvider
}

func (*YouTubeProvider) getInfoPlaceholder(id string) types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title: "YouTube " + strings.TrimPrefix(id, internal_id.YtInternalPrefix),
	}
}

func (p *YouTubeProvider) getInfo(id string, ctx context.Context) (types.GeneralVideoInfo, error) {
	if p.currentMode() == ModeDisabled {
		return types.GeneralVideoInfo{}, errYouTubeDisabled
	}
	if !lo.Contains(p.currentAllowResources(), ResourceInfo) {
		return types.GeneralVideoInfo{}, errYouTubeInfoDisabled
	}

	if videoID, isYoutube := internal_id.CheckIdIsYoutube(id); isYoutube {
		if p.currentMode() == ModeApi {
			if key := secrets.Get(secrets.YoutubeKeyUser); key != "" {
				info, err := api.GetYoutubeInfo(videoID, key, ctx)
				if err != nil {
					return types.GeneralVideoInfo{}, fmt.Errorf("get YouTube info from api: %w", err)
				}

				return *info, nil
			}

			return types.GeneralVideoInfo{}, errYouTubeApiNotConfigured
		}

		if p.currentMode() == ModeYtDlp {
			info, err := local_executables.GetVideoBasicInfoWithYtDlp(internal_id.GetStandardYoutubeURL(videoID), ctx)
			if err != nil {
				return types.GeneralVideoInfo{}, fmt.Errorf("get YouTube info by yt-dlp: %w", err)
			}

			return *info, nil
		}
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *YouTubeProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if p.currentMode() == ModeDisabled {
		return nil, errYouTubeDisabled
	}
	if !lo.Contains(p.currentAllowResources(), ResourceThumbnail) {
		return nil, errYouTubeThumbnailDisabled
	}

	if videoID, isYoutube := internal_id.CheckIdIsYoutube(id); isYoutube {
		i, err := GetThumbnailImage(requesting.GetClient(requesting.YouTubeImage), internal_id.GetYoutubeMQThumbnailURL(videoID), ctx)
		if err != nil {
			return nil, fmt.Errorf("get YouTube thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *YouTubeProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if p.currentMode() == ModeDisabled {
		return nil, errYouTubeDisabled
	}
	if p.currentMode() == ModeApi {
		return nil, errYouTubeApiNotSupported
	}
	if !lo.Contains(p.currentAllowResources(), ResourceVideo) {
		return nil, errYouTubeVideoDisabled
	}

	if videoID, isYoutube := internal_id.CheckIdIsYoutube(id); isYoutube {
		return ytDlpResolve(internal_id.GetStandardYoutubeURL(videoID), requesting.GetClient(requesting.YouTubeVideo), ctx)
	}

	return nil, ErrUnexpectedParam
}

func (p *YouTubeProvider) loop() {
	modeCh := p.modeEm.SubscribeEvent()
	defer modeCh.Close()
	resourcesCh := p.allowedResourcesEm.SubscribeEvent()
	defer resourcesCh.Close()
	keySource := secrets.Source(secrets.YoutubeKeyUser)
	defer keySource.Close()

	apiClientCh := requesting.GetClient(requesting.YouTubeApi).SubscribeChange()
	defer apiClientCh.Close()
	thumbnailClientCh := requesting.GetClient(requesting.YouTubeImage).SubscribeChange()
	defer thumbnailClientCh.Close()
	videoClientCh := requesting.GetClient(requesting.YouTubeVideo).SubscribeChange()
	defer videoClientCh.Close()

	// A watcher of the availability level: it is read again whenever it wakes the
	// loop, so a dropped or late wake-up cannot leave the gates stale
	// (review/09 §2.5).
	ytdlpAvailability := local_executables.YtDlpAvailability()
	defer ytdlpAvailability.Close()

	p.refreshGates()

	for {
		select {
		case <-p.stopCh:
			return
		// Every input the gates derive from: the fetching mode, the resource
		// list, the api key and the yt-dlp binary. All three gates are
		// recomputed from the current state instead of from a copy kept here.
		case <-modeCh.Channel:
			p.refreshGates()
		case <-resourcesCh.Channel:
			p.refreshGates()
		case <-keySource.Wakes():
			p.refreshGates()
		case <-ytdlpAvailability.Wakes():
			p.refreshGates()
		// A new HTTP client replaced the previous one: the entries that failed
		// with the old one are worth another try.
		case <-apiClientCh.Channel:
			p.infoAvailable.Retry()
		case <-thumbnailClientCh.Channel:
			p.thumbnailAvailable.Retry()
		case <-videoClientCh.Channel:
			p.videoAvailable.Retry()
		}
	}
}

// refreshGates recomputes YouTube's three gates from the current mode, the api
// key, yt-dlp and the resource list.
func (p *YouTubeProvider) refreshGates() {
	resources := p.currentAllowResources()
	apiReady := p.currentMode() == ModeApi && secrets.Get(secrets.YoutubeKeyUser) != ""
	ytdlpReady := p.currentMode() == ModeYtDlp && local_executables.YtDlpAvailable()

	// The api answers info requests, yt-dlp answers both info and video ones,
	// and thumbnails come from YouTube's image CDN in either mode.
	p.updateGate(p.infoAvailable, resources, ResourceInfo, ResourceAvailable(apiReady || ytdlpReady))
	p.updateGate(p.thumbnailAvailable, resources, ResourceThumbnail, true)
	p.updateGate(p.videoAvailable, resources, ResourceVideo, ResourceAvailable(ytdlpReady))
}

func (p *YouTubeProvider) Start() {
	p.wg.Go(p.loop)
}

func newYouTubeProvider() ResourceProvider {
	p := &YouTubeProvider{}
	p.PlatformProvider = constructPlatformProvider(p)
	p.setup("YouTube")

	return p
}

// BiliBili

var (
	errBiliBiliDisabled          = fmt.Errorf("%w: fetching BiliBili", ErrFeatureDisabled)
	errBiliBiliInfoDisabled      = fmt.Errorf("%w: fetching BiliBili info", ErrFeatureDisabled)
	errBiliBiliThumbnailDisabled = fmt.Errorf("%w: fetching BiliBili thumbnail", ErrFeatureDisabled)
	errBiliBiliVideoDisabled     = fmt.Errorf("%w: fetching BiliBili video", ErrFeatureDisabled)
)

type BiliBiliProvider struct {
	PlatformProvider
}

func (*BiliBiliProvider) getInfoPlaceholder(id string) types.GeneralVideoInfo {
	return types.GeneralVideoInfo{
		Title: "BiliBili " + strings.TrimPrefix(id, internal_id.BiliInternalPrefix),
	}
}

func (p *BiliBiliProvider) getInfo(id string, ctx context.Context) (types.GeneralVideoInfo, error) {
	if p.currentMode() == ModeDisabled {
		return types.GeneralVideoInfo{}, errBiliBiliDisabled
	}
	if !lo.Contains(p.currentAllowResources(), ResourceInfo) {
		return types.GeneralVideoInfo{}, errBiliBiliInfoDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.currentMode() == ModeApi {
			info, err := api.GetBiliBiliInfo(bvId, ctx)
			if err != nil {
				return types.GeneralVideoInfo{}, fmt.Errorf("get BiliBili info from api: %w", err)
			}

			return *info, nil
		}

		if p.currentMode() == ModeYtDlp {
			info, err := local_executables.GetVideoBasicInfoWithYtDlp(internal_id.GetStandardBiliURL(bvId), ctx)
			if err != nil {
				return types.GeneralVideoInfo{}, fmt.Errorf("get BiliBili info by yt-dlp: %w", err)
			}

			return *info, nil
		}
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *BiliBiliProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if p.currentMode() == ModeDisabled {
		return nil, errBiliBiliDisabled
	}
	if !lo.Contains(p.currentAllowResources(), ResourceThumbnail) {
		return nil, errBiliBiliThumbnailDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.currentMode() == ModeApi {
			info, err := api.GetBvInfo(bvId, ctx)
			if err != nil {
				return nil, fmt.Errorf("get BiliBili thumbnail from api: %w", err)
			}

			return GetThumbnailImage(requesting.GetClient(requesting.BiliBili), info.Pic, ctx)
		}

		if p.currentMode() == ModeYtDlp {
			url, err := local_executables.GetVideoThumbnailWithYtDlp(internal_id.GetStandardBiliURL(bvId), ctx)
			if err != nil {
				return nil, fmt.Errorf("get BiliBili thumbnail by yt-dlp: %w", err)
			}

			return GetThumbnailImage(requesting.GetClient(requesting.BiliBili), url, ctx)
		}
	}

	return nil, ErrUnexpectedParam
}

func (p *BiliBiliProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if p.currentMode() == ModeDisabled {
		return nil, errBiliBiliDisabled
	}
	if !lo.Contains(p.currentAllowResources(), ResourceVideo) {
		return nil, errBiliBiliVideoDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.currentMode() == ModeApi {
			url, lastModified, err := api.GetBiliBiliUrlAndUploadTime(bvId, ctx)

			info, err := directResolve(url, requesting.GetClient(requesting.BiliBili), ctx)
			if err != nil {
				return nil, err
			}

			info.LastModified = lastModified
			return info, nil
		}
		if p.currentMode() == ModeYtDlp {
			return ytDlpResolve(internal_id.GetStandardBiliURL(bvId), requesting.GetClient(requesting.BiliBili), ctx)
		}
	}

	return nil, ErrUnexpectedParam
}

func (p *BiliBiliProvider) loop() {
	modeCh := p.modeEm.SubscribeEvent()
	defer modeCh.Close()
	resourcesCh := p.allowedResourcesEm.SubscribeEvent()
	defer resourcesCh.Close()

	clientCh := requesting.GetClient(requesting.BiliBili).SubscribeChange()
	defer clientCh.Close()

	ytdlpAvailability := local_executables.YtDlpAvailability()
	defer ytdlpAvailability.Close()

	p.refreshGates()

	for {
		select {
		case <-p.stopCh:
			return
		case <-modeCh.Channel:
			p.refreshGates()
		case <-resourcesCh.Channel:
			p.refreshGates()
		case <-ytdlpAvailability.Wakes():
			p.refreshGates()
		// A new HTTP client replaced the previous one: the entries that failed
		// with the old one are worth another try.
		case <-clientCh.Channel:
			p.retryGates()
		}
	}
}

// refreshGates recomputes BiliBili's three gates: the api mode answers all three
// resources, the yt-dlp mode answers them as long as the binary is there.
func (p *BiliBiliProvider) refreshGates() {
	resources := p.currentAllowResources()
	available := ResourceAvailable(
		p.currentMode() == ModeApi ||
			(p.currentMode() == ModeYtDlp && local_executables.YtDlpAvailable()),
	)

	p.updateGate(p.infoAvailable, resources, ResourceInfo, available)
	p.updateGate(p.thumbnailAvailable, resources, ResourceThumbnail, available)
	p.updateGate(p.videoAvailable, resources, ResourceVideo, available)
}

func (p *BiliBiliProvider) Start() {
	p.wg.Go(p.loop)
}

func newBiliBiliProvider() ResourceProvider {
	p := &BiliBiliProvider{}
	p.PlatformProvider = constructPlatformProvider(p)
	p.setup("BiliBili")

	return p
}
