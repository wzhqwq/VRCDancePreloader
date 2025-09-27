package song

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var idIncrement int64 = 0

var logger = utils.NewUniqueLogger()

type PreloadedSong struct {
	sm         *StateMachine
	PyPySong   *raw_song.PyPyDanceSong
	CustomSong *raw_song.CustomSong
	WannaSong  *raw_song.WannaDanceSong

	InfoNa bool

	// constant
	Adder   string
	Unknown bool
	ID      int64

	// play progress states
	Duration   time.Duration
	TimePassed time.Duration

	// download progress states
	TotalSize      int64
	DownloadedSize int64

	// diagnostic states
	PreloadError error

	// event
	em     *utils.EventManager[ChangeType]
	lazyEm *utils.EventManager[ChangeType]
}

// getters

func (ps *PreloadedSong) GetOriginalUrl() string {
	if ps.PyPySong != nil && len(ps.PyPySong.OriginalURL) > 0 {
		return ps.PyPySong.OriginalURL[0]
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetDownloadUrl() string {
	if ps.PyPySong != nil {
		return utils.GetPyPyVideoUrl(ps.PyPySong.ID)
	}
	if ps.WannaSong != nil {
		return utils.GetWannaVideoUrl(ps.WannaSong.DanceId)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.Url
	}
	return ""
}
func (ps *PreloadedSong) GetSongId() string {
	if ps.Unknown {
		// TODO unique id for unknown song
		return "unknown"
	}
	if ps.PyPySong != nil {
		return fmt.Sprintf("pypy_%d", ps.PyPySong.ID)
	}
	if ps.WannaSong != nil {
		return fmt.Sprintf("wanna_%d", ps.WannaSong.DanceId)
	}
	if ps.CustomSong != nil {
		return ps.CustomSong.UniqueId
	}
	return "unknown"
}
func (ps *PreloadedSong) GetPreloadStatus() DownloadStatus {
	return ps.sm.DownloadStatus
}
func (ps *PreloadedSong) DownloadInstantly(complete bool, ctx context.Context) (cache.Entry, error) {
	err := ps.sm.DownloadInstantly(complete)
	if err != nil {
		return nil, err
	}

	logPrefix := ""
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		logPrefix = "[" + traceID + "]"
	}

	// reference the cache entry until request closed
	entry, err := cache.OpenCacheEntry(ps.GetSongId(), logPrefix)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		cache.ReleaseCacheEntry(ps.GetSongId(), logPrefix)
	}()

	return entry, nil
}

// compare

func (ps *PreloadedSong) MatchWithCustomUrl(url string) bool {
	if ps.CustomSong == nil {
		return false
	}
	return ps.CustomSong.MatchUrl(url)
}
func (ps *PreloadedSong) MatchWithPyPyId(id int) bool {
	if ps.PyPySong == nil {
		return false
	}
	return ps.PyPySong.ID == id
}
func (ps *PreloadedSong) MatchWithWannaId(id int) bool {
	if ps.WannaSong == nil {
		return false
	}
	return ps.WannaSong.DanceId == id
}
func (ps *PreloadedSong) Match(another *PreloadedSong) bool {
	if ps.PyPySong != nil && another.PyPySong != nil {
		return ps.PyPySong.ID == another.PyPySong.ID
	}
	if ps.WannaSong != nil && another.WannaSong != nil {
		return ps.WannaSong.DanceId == another.WannaSong.DanceId
	}
	if ps.CustomSong != nil && another.CustomSong != nil {
		return ps.CustomSong.UniqueId == another.CustomSong.UniqueId
	}
	return false
}

// actions

func (ps *PreloadedSong) PlaySongStartFrom(offset time.Duration) bool {
	if ps.Duration > 1 {
		ps.sm.PlaySongStartFrom(offset)
		return true
	}
	ps.completeDuration()
	return false
}
func (ps *PreloadedSong) CancelPlaying() {
	ps.sm.CancelPlayingLoop()
}

func (ps *PreloadedSong) PreloadSong() {
	if ps.sm.CanPreload() {
		go func() {
			ps.sm.reserveForPreload()
			ps.sm.StartDownload()
		}()
	}
}
func (ps *PreloadedSong) PrioritizeSong() {
	ps.sm.Prioritize()
}
func (ps *PreloadedSong) InDownloadQueue() bool {
	return ps.sm.IsDownloadLoopStarted()
}
func (ps *PreloadedSong) RemoveFromList() {
	ps.sm.RemoveFromList()
}
func (ps *PreloadedSong) AddToHistory() {
	info := ps.GetInfo()
	startTime := time.Now().Add(-ps.TimePassed).Unix()
	persistence.AddToHistory(info.ID, info.Title, ps.Adder, time.Unix(startTime, 0))
}
