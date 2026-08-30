package song

import (
	"context"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/persistence"
	"github.com/wzhqwq/VRCDancePreloader/internal/types"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var songLogger = utils.NewLogger("Song")
var activeSongLogger = utils.NewLogger("Song (Active)")
var removedSongLogger = utils.NewLogger("Song (Removed)")
var resetSongLogger = utils.NewLogger("Song (Reset)")

type StatefulSong struct {
	sm *StateMachine

	songId string
	info   types.GeneralVideoInfo

	// constant
	Adder   string
	Unknown bool
	ID      int64

	// play progress states
	TimePassed time.Duration

	// download progress states
	TotalSize      int64
	DownloadedSize int64

	// diagnostic states
	PreloadError error

	stopCh chan struct{}

	// event
	em     *utils.EventManager[ChangeType]
	lazyEm *utils.EventManager[ChangeType]
}

// getters

func (ps *StatefulSong) SongId() string {
	if ps.Unknown {
		// TODO unique id for unknown song
		return "unknown"
	}

	return ps.songId
}

func (ps *StatefulSong) EstDuration() time.Duration {
	if ps.info.Duration == 0 {
		return time.Minute
	}
	return ps.info.Duration
}

func (ps *StatefulSong) StateMachine() *StateMachine {
	return ps.sm
}

func (ps *StatefulSong) PreloadStatus() DownloadStatus {
	return ps.sm.DownloadStatus
}

func (ps *StatefulSong) ValidLogger(ctx context.Context) utils.LoggerImpl {
	var logger = songLogger
	if reqLogger, ok := ctx.Value("logger").(*utils.CustomLogger); ok {
		logger = reqLogger
	}
	return logger
}

// compare

func (ps *StatefulSong) MatchWithCustomUrl(url string) bool {
	return ps.songId == raw_song.GetInternalIdByUrl(url)
}
func (ps *StatefulSong) Match(another *StatefulSong) bool {
	return ps.songId == another.songId
}

// actions

func (ps *StatefulSong) Destroy() {
	close(ps.stopCh)
	ps.sm.Destroy()
}
func (ps *StatefulSong) AddToHistory() {
	info := ps.GetInfo()
	startTime := time.Now().Add(-ps.TimePassed).Unix()
	persistence.AddToHistory(info.ID, info.Title, ps.Adder, time.Unix(startTime, 0))
}
