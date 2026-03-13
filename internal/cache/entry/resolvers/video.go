package resolvers

import (
	"context"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache/entry"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/local_executables"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DuDuResolver struct {
	entry.DirectUrlResolver
	ID int
}

func (r *DuDuResolver) Resolve(logger utils.LoggerImpl, ctx context.Context) (*entry.RemoteVideoInfo, error) {
	info, err := r.DirectUrlResolver.Resolve(logger, ctx)
	if err != nil {
		return nil, err
	}

	if song, ok := raw_song.FindDuDuSong(r.ID); ok && song.PublishedAt > 0 {
		info.LastModified = time.Unix(int64(song.PublishedAt), 0)
	}
	return info, nil
}

type YtDlpResolver struct {
	entry.DirectUrlResolver
	url string
}

func (r *YtDlpResolver) Resolve(logger utils.LoggerImpl, ctx context.Context) (*entry.RemoteVideoInfo, error) {
	logger.InfoLn("Resolving", r.url, "using yt-dlp")
	url, err := local_executables.ResolveVideoUrlWithYtDlp(r.url, ctx)
	if err != nil {
		return nil, err
	}

	return r.DirectUrlResolver.SetUrl(url).Resolve(logger, ctx)
}

func (r *YtDlpResolver) Check(hintUrl string) bool {
	if r.url == "" {
		r.url = hintUrl
	}
	return true
}

func newYtDlpResolver(url string, finalClient *requesting.ClientProvider) *YtDlpResolver {
	return &YtDlpResolver{
		DirectUrlResolver: entry.ConstructDirectUrlResolver("", finalClient),

		url: url,
	}
}

type BiliBiliResolver struct {
	entry.DirectUrlResolver
	ID string
}

func (r *BiliBiliResolver) Resolve(logger utils.LoggerImpl, ctx context.Context) (*entry.RemoteVideoInfo, error) {
	logger.InfoLn("Resolving BiliBili video", r.ID, "using API")
	mTime, err := third_party_api.GetBiliVideoModTime(r.ID, ctx)
	if err != nil {
		return nil, err
	}

	url, err := third_party_api.GetBiliVideoUrl(r.ID, ctx)
	if err != nil {
		return nil, err
	}

	info, err := r.DirectUrlResolver.SetUrl(url).Resolve(logger, ctx)
	if err != nil {
		return nil, err
	}

	info.LastModified = mTime

	return info, nil
}

func InitThirdPartyVideoProviders() {
	entry.SetResolverProvider(func(id string) entry.UrlResolver {
		if num, ok := utils.CheckIdIsPyPy(id); ok {
			originalUrl := ""
			if song, ok := raw_song.FindPyPySong(num); ok && len(song.OriginalURL) > 0 {
				originalUrl = song.OriginalURL[0]
			}
			return entry.NewDirectUrlResolver(
				utils.GetPyPyVideoUrl(num),
				requesting.GetClient(requesting.PyPyDance),
				// fallback using yt-dlp
				newYtDlpResolver(originalUrl, requesting.GetClient(requesting.YouTubeVideo)),
			)
		}
		if num, ok := utils.CheckIdIsWanna(id); ok {
			return entry.NewDirectUrlResolver(
				utils.GetWannaVideoUrl(num),
				requesting.GetClient(requesting.WannaDance),
			)
		}
		if num, ok := utils.CheckIdIsDuDu(id); ok {
			return &DuDuResolver{
				DirectUrlResolver: entry.ConstructDirectUrlResolver(
					utils.GetDuDuVideoUrl(num),
					requesting.GetClient(requesting.DuDuFitDance),
				),
				ID: num,
			}
		}
		if bvID, ok := utils.CheckIdIsBili(id); ok {
			return &BiliBiliResolver{
				DirectUrlResolver: entry.ConstructDirectUrlResolver(
					"",
					requesting.GetClient(requesting.BiliBiliApi),
					// fallback using yt-dlp
					newYtDlpResolver(utils.GetStandardBiliURL(bvID), requesting.GetClient(requesting.BiliBiliVideo)),
				),
				ID: bvID,
			}
		}
		if ytID, ok := utils.CheckIdIsYoutube(id); ok {
			return newYtDlpResolver(utils.GetStandardYoutubeURL(ytID), requesting.GetClient(requesting.YouTubeVideo))
		}
		return nil
	})
}
