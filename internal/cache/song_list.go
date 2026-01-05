package cache

import (
	"context"

	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var pypyListResource *utils.RemoteResource[raw_song.PyPyDanceListResponse]
var wannaListResource *utils.RemoteResource[raw_song.WannaDanceListResponse]
var duduListResource *utils.RemoteResource[raw_song.DuDuFitDanceListResponse]

var songCtx context.Context

func InitSongList(ctx context.Context) {
	pypyListResource = utils.NewJsonRemoteResource[raw_song.PyPyDanceListResponse](
		"PyPyDance Manifest", utils.GetPyPyListUrl(), requesting.GetPyPyClient,
	)
	wannaListResource = utils.NewJsonRemoteResource[raw_song.WannaDanceListResponse](
		"WannaDance Manifest", utils.GetWannaListUrl(), requesting.GetWannaClient,
	)
	duduListResource = utils.NewJsonRemoteResource[raw_song.DuDuFitDanceListResponse](
		"DuDuFitDance Manifest", utils.GetDuDuListUrl(), requesting.GetDuDuClient,
	)
	songCtx = ctx
	DownloadPyPySongs()
	DownloadWannaSongs()
	DownloadDuDuSongs()
}

func DownloadPyPySongs() {
	go func() {
		if pypyListResource.StartDownload(songCtx) {
			raw_song.ProcessPyPyDanceList(pypyListResource.Get())
		}
	}()
}

func DownloadWannaSongs() {
	go func() {
		if wannaListResource.StartDownload(songCtx) {
			raw_song.ProcessWannaDanceList(wannaListResource.Get())
		}
	}()
}

func DownloadDuDuSongs() {
	go func() {
		if duduListResource.StartDownload(songCtx) {
			raw_song.ProcessDuDuFitDanceList(duduListResource.Get())
		}
	}()
}
