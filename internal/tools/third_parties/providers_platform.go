package third_parties

import (
	"context"
	"errors"
	"fmt"
	"image"
	"strings"

	"github.com/samber/lo"
	"github.com/wzhqwq/VRCDancePreloader/internal/gui/images/thumbnails"
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

	mode string

	modeEm *utils.EventManager[string]

	infoAvailableEm *utils.EventManager[bool]

	infoAvailable, videoAvailable, thumbnailAvailable *BothTrue
}

func ValidMode(mode string) bool {
	return mode == ModeDisabled || mode == ModeApi || mode == ModeYtDlp
}

func (p *PlatformProvider) SetMode(mode string) {
	if mode == p.mode {
		return
	}
	p.mode = mode
	p.modeEm.NotifySubscribers(mode)
}

func (p *PlatformProvider) setup(name string) {
	p.BaseProvider.setup(name)
	p.infoManager.BindScheduler(utils.SharedVideoScheduler())
	p.infoManager.BindAvailability(p.infoAvailableEm.SubscribeEvent)
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
		modeEm:          utils.NewEventManager[string](),
		BaseProvider:    constructBaseProvider(p, 1),
		infoAvailableEm: utils.NewEventManager[bool](),
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
	if p.mode == ModeDisabled {
		return types.GeneralVideoInfo{}, errYouTubeDisabled
	}
	if !lo.Contains(p.allowResources, ResourceInfo) {
		return types.GeneralVideoInfo{}, errYouTubeInfoDisabled
	}

	if videoID, isYoutube := internal_id.CheckIdIsYoutube(id); isYoutube {
		if p.mode == ModeApi {
			if key := secrets.Get(secrets.YoutubeKeyUser); key != "" {
				info, err := api.GetYoutubeInfo(videoID, key, ctx)
				if err != nil {
					return types.GeneralVideoInfo{}, fmt.Errorf("get YouTube info from api: %w", err)
				}

				return *info, nil
			}

			return types.GeneralVideoInfo{}, errYouTubeApiNotConfigured
		}

		if p.mode == ModeYtDlp {
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
	if p.mode == ModeDisabled {
		return nil, errYouTubeDisabled
	}
	if !lo.Contains(p.allowResources, ResourceThumbnail) {
		return nil, errYouTubeThumbnailDisabled
	}

	if videoID, isYoutube := internal_id.CheckIdIsYoutube(id); isYoutube {
		i, err := thumbnails.GetThumbnailImage(requesting.GetClient(requesting.YouTubeImage), internal_id.GetYoutubeMQThumbnailURL(videoID), ctx)
		if err != nil {
			return nil, fmt.Errorf("get YouTube thumbnail: %w", err)
		}

		return i, nil
	}

	return nil, ErrUnexpectedParam
}

func (p *YouTubeProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if p.mode == ModeDisabled {
		return nil, errYouTubeDisabled
	}
	if p.mode == ModeApi {
		return nil, errYouTubeApiNotSupported
	}
	if !lo.Contains(p.allowResources, ResourceVideo) {
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
	keyCh := secrets.Subscribe(secrets.YoutubeKeyUser)
	defer keyCh.Close()

	apiClientCh := requesting.GetClient(requesting.YouTubeApi).SubscribeChange()
	defer apiClientCh.Close()
	thumbnailClientCh := requesting.GetClient(requesting.YouTubeImage).SubscribeChange()
	defer thumbnailClientCh.Close()
	videoClientCh := requesting.GetClient(requesting.YouTubeVideo).SubscribeChange()
	defer videoClientCh.Close()

	ytdlpAvailableCh := local_executables.SubscribeYtDlpAvailability()
	defer ytdlpAvailableCh.Close()

	apiAvailable := p.mode == ModeApi && secrets.Get(secrets.YoutubeKeyUser) != ""
	ytdlpAvailable := p.mode == ModeYtDlp && local_executables.YtDlpAvailable()

	infoAllowed := lo.Contains(p.allowResources, ResourceInfo)
	thumbnailAllowed := lo.Contains(p.allowResources, ResourceThumbnail)
	videoAllowed := lo.Contains(p.allowResources, ResourceVideo)

	p.infoAvailable = newBothTrue(apiAvailable || ytdlpAvailable, infoAllowed, p.infoAvailableEm)
	p.thumbnailAvailable = newBothTrue(true, thumbnailAllowed, p.thumbnailAvailableEm)
	p.videoAvailable = newBothTrue(ytdlpAvailable, videoAllowed, p.videoAvailableEm)

	for {
		select {
		case <-p.stopCh:
			return
		// fetching mode changed
		case mode := <-modeCh.Channel:
			apiAvailable = mode == ModeApi && secrets.Get(secrets.YoutubeKeyUser) != ""
			ytdlpAvailable = mode == ModeYtDlp && local_executables.YtDlpAvailable()

			p.infoAvailable.SetAvailable(apiAvailable || ytdlpAvailable)
			p.videoAvailable.SetAvailable(ytdlpAvailable)
		// allowed resources changed
		case allowed := <-resourcesCh.Channel:
			infoAllowed = lo.Contains(allowed, ResourceInfo)
			thumbnailAllowed = lo.Contains(allowed, ResourceThumbnail)
			videoAllowed = lo.Contains(allowed, ResourceVideo)

			p.infoAvailable.SetAllowed(infoAllowed)
			p.thumbnailAvailable.SetAllowed(thumbnailAllowed)
			p.videoAvailable.SetAllowed(videoAllowed)
		// key changed
		case key := <-keyCh.Channel:
			apiAvailable = p.mode == ModeApi && key != ""

			p.infoAvailable.SetAvailable(apiAvailable || ytdlpAvailable)
		// client changed
		case <-apiClientCh.Channel:
			p.infoAvailable.NotifyIfSatisfied()
		case <-thumbnailClientCh.Channel:
			p.thumbnailAvailable.NotifyIfSatisfied()
		case <-videoClientCh.Channel:
			p.videoAvailable.NotifyIfSatisfied()
		// YtDlp availability changed
		case ok := <-ytdlpAvailableCh.Channel:
			ytdlpAvailable = p.mode == ModeYtDlp && ok

			p.infoAvailable.SetAvailable(apiAvailable || ytdlpAvailable)
			p.videoAvailable.SetAvailable(ytdlpAvailable)
		}
	}
}

func newYouTubeProvider() ResourceProvider {
	p := &YouTubeProvider{}
	p.PlatformProvider = constructPlatformProvider(p)
	p.setup("YouTube")

	p.wg.Go(p.loop)
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
	if p.mode == ModeDisabled {
		return types.GeneralVideoInfo{}, errBiliBiliDisabled
	}
	if !lo.Contains(p.allowResources, ResourceInfo) {
		return types.GeneralVideoInfo{}, errBiliBiliInfoDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.mode == ModeApi {
			info, err := api.GetBiliBiliInfo(bvId, ctx)
			if err != nil {
				return types.GeneralVideoInfo{}, fmt.Errorf("get BiliBili info from api: %w", err)
			}

			return *info, nil
		}

		if p.mode == ModeYtDlp {
			info, err := local_executables.GetVideoBasicInfoWithYtDlp(internal_id.GetStandardYoutubeURL(bvId), ctx)
			if err != nil {
				return types.GeneralVideoInfo{}, fmt.Errorf("get BiliBili info by yt-dlp: %w", err)
			}

			return *info, nil
		}
	}

	return types.GeneralVideoInfo{}, ErrUnexpectedParam
}

func (p *BiliBiliProvider) getThumbnail(id string, ctx context.Context) (image.Image, error) {
	if p.mode == ModeDisabled {
		return nil, errBiliBiliDisabled
	}
	if !lo.Contains(p.allowResources, ResourceThumbnail) {
		return nil, errBiliBiliThumbnailDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.mode == ModeApi {
			info, err := api.GetBvInfo(bvId, ctx)
			if err != nil {
				return nil, fmt.Errorf("get BiliBili thumbnail from api: %w", err)
			}

			return thumbnails.GetThumbnailImage(requesting.GetClient(requesting.BiliBili), info.Pic, ctx)
		}

		if p.mode == ModeYtDlp {
			url, err := local_executables.GetVideoThumbnailWithYtDlp(internal_id.GetStandardBiliURL(bvId), ctx)
			if err != nil {
				return nil, fmt.Errorf("get BiliBili thumbnail by yt-dlp: %w", err)
			}

			return thumbnails.GetThumbnailImage(requesting.GetClient(requesting.BiliBili), url, ctx)
		}
	}

	return nil, ErrUnexpectedParam
}

func (p *BiliBiliProvider) resolveVideoUrl(id string, ctx context.Context) (*types.RemoteHttpResourceInfo, error) {
	if p.mode == ModeDisabled {
		return nil, errBiliBiliDisabled
	}
	if !lo.Contains(p.allowResources, ResourceVideo) {
		return nil, errBiliBiliVideoDisabled
	}

	if bvId, isBiliBili := internal_id.CheckIdIsBili(id); isBiliBili {
		if p.mode == ModeApi {
			url, lastModified, err := api.GetBiliBiliUrlAndUploadTime(bvId, ctx)

			info, err := directResolve(url, requesting.GetClient(requesting.BiliBili), ctx)
			if err != nil {
				return nil, err
			}

			info.LastModified = lastModified
			return info, nil
		}
		if p.mode == ModeYtDlp {
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

	ytdlpAvailableCh := local_executables.SubscribeYtDlpAvailability()
	defer ytdlpAvailableCh.Close()

	available := p.mode == ModeApi || (p.mode == ModeYtDlp && local_executables.YtDlpAvailable())

	infoAllowed := lo.Contains(p.allowResources, ResourceInfo)
	thumbnailAllowed := lo.Contains(p.allowResources, ResourceThumbnail)
	videoAllowed := lo.Contains(p.allowResources, ResourceVideo)

	p.infoAvailable = newBothTrue(available, infoAllowed, p.infoAvailableEm)
	p.thumbnailAvailable = newBothTrue(available, thumbnailAllowed, p.thumbnailAvailableEm)
	p.videoAvailable = newBothTrue(available, videoAllowed, p.videoAvailableEm)

	for {
		select {
		case <-p.stopCh:
			return
		// fetching mode changed
		case mode := <-modeCh.Channel:
			available = mode == ModeApi || (mode == ModeYtDlp && local_executables.YtDlpAvailable())

			p.infoAvailable.SetAvailable(available)
			p.thumbnailAvailable.SetAvailable(available)
			p.videoAvailable.SetAvailable(available)
		case allowed := <-resourcesCh.Channel:
			infoAllowed = lo.Contains(allowed, ResourceInfo)
			thumbnailAllowed = lo.Contains(allowed, ResourceThumbnail)
			videoAllowed = lo.Contains(allowed, ResourceVideo)

			p.infoAvailable.SetAllowed(infoAllowed)
			p.thumbnailAvailable.SetAllowed(thumbnailAllowed)
			p.videoAvailable.SetAllowed(videoAllowed)
		// client changed
		case <-clientCh.Channel:
			p.infoAvailable.NotifyIfSatisfied()
			p.thumbnailAvailable.NotifyIfSatisfied()
			p.videoAvailable.NotifyIfSatisfied()
		// YtDlp availability changed
		case ok := <-ytdlpAvailableCh.Channel:
			available = p.mode == ModeApi || (p.mode == ModeYtDlp && ok)

			p.infoAvailable.SetAvailable(available)
			p.thumbnailAvailable.SetAvailable(available)
			p.videoAvailable.SetAvailable(available)
		}
	}
}

func newBiliBiliProvider() ResourceProvider {
	p := &BiliBiliProvider{}
	p.PlatformProvider = constructPlatformProvider(p)
	p.setup("BiliBili")

	p.wg.Go(p.loop)
	return p
}
