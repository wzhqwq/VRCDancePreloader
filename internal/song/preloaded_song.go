package song

import (
	"context"
	"fmt"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/cache"
	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var songLogger = utils.NewLogger("Song")
var activeSongLogger = utils.NewLogger("Song (Active)")
var removedSongLogger = utils.NewLogger("Song (Removed)")

type PreloadedSong struct {
	sm         *StateMachine
	PyPySong   *raw_song.PyPyDanceSong
	CustomSong *raw_song.CustomSong
	WannaSong  *raw_song.WannaDanceSong
	DuDuSong   *raw_song.DuDuFitDanceSong

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
	if ps.DuDuSong != nil {
		return fmt.Sprintf("dudu_%d", ps.DuDuSong.ID)
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

	var logger = songLogger
	if reqLogger, ok := ctx.Value("logger").(*utils.CustomLogger); ok {
		logger = reqLogger
	}

	// reference the cache entry until request closed
	entry, err := cache.OpenCacheEntry(ps.GetSongId(), logger)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		cache.ReleaseCacheEntry(ps.GetSongId(), logger)
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
func (ps *PreloadedSong) MatchWithDuDuId(id int) bool {
	if ps.DuDuSong == nil {
		return false
	}
	return ps.DuDuSong.ID == id
}
func (ps *PreloadedSong) Match(another *PreloadedSong) bool {
	if ps.PyPySong != nil && another.PyPySong != nil {
		return ps.PyPySong.ID == another.PyPySong.ID
	}
	if ps.WannaSong != nil && another.WannaSong != nil {
		return ps.WannaSong.DanceId == another.WannaSong.DanceId
	}
	if ps.DuDuSong != nil && another.DuDuSong != nil {
		return ps.DuDuSong.ID == another.DuDuSong.ID
	}
	if ps.CustomSong != nil && another.CustomSong != nil {
		return ps.CustomSong.UniqueId == another.CustomSong.UniqueId
	}
	return false
}

// actions

func (ps *PreloadedSong) PlaySongStartFrom(offset time.Duration) bool {
	if ps.Duration > 1 {
		ps.sm.PlaySongAndSync(offset)
		return true
	}
	ps.completeDuration()
	return false
}
func (ps *PreloadedSong) PlaySong() {
	ps.sm.PlaySong()
	if ps.Duration == 0 {
		ps.completeDuration()
	}
}
func (ps *PreloadedSong) CancelPlaying() {
	ps.sm.CancelPlayingLoop()
}

func (ps *PreloadedSong) PreloadSong() {
	if ps.sm.CanPreload() {
		ps.sm.StartDownload()
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
func (ps *PreloadedSong) UpdateStartPlayingEta(eta time.Duration) {
	if ps.sm.DownloadStatus == Downloading {
		download.UpdateRequestEta(ps.GetSongId(), time.Now().Add(eta), ps.Duration)
	}
}
func (ps *PreloadedSong) AddToHistory() {
	info := ps.GetInfo()
	startTime := time.Now().Add(-ps.TimePassed).Unix()
	persistence.AddToHistory(info.ID, info.Title, ps.Adder, time.Unix(startTime, 0))
}
