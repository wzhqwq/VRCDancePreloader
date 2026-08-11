package song

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

// Info, immutable

type PreloadedSongInfo struct {
	ID    string
	Title string
	Group string
	Adder string
	Size  string
}

func (ps *StatefulSong) GetInfo() (basicInfo PreloadedSongInfo) {
	adder := i18n.T("wrapper_adder", goeasyi18n.Options{
		Data: map[string]any{"Adder": ps.Adder},
	})
	if ps.Adder == "" {
		adder = i18n.T("placeholder_unknown_adder")
	}
	if ps.Adder == "Random" || ps.Adder == "PyPyDance" {
		adder = i18n.T("placeholder_random_play")
	}

	size := i18n.T("placeholder_unknown_size")
	if ps.TotalSize > 0 {
		size = utils.PrettyByteSize(ps.TotalSize)
	}

	basicInfo = PreloadedSongInfo{
		ID: ps.SongId(),

		Adder: adder,
		Size:  size,
	}

	if ps.Unknown {
		basicInfo.Title = i18n.T("placeholder_unknown_song")
		basicInfo.Group = i18n.T("placeholder_custom_song")
		return
	}

	if ps.info.Title != "" {
		basicInfo.Title = ps.info.Title
		basicInfo.Group = ps.info.GroupName
		if basicInfo.Group == "" {
			basicInfo.Group = internal_id.GetPlatformNameByInternalId(ps.songId)
		}
		return
	}

	basicInfo.Title = "URL: " + raw_song.GetUrlByInternalId(ps.songId)
	basicInfo.Group = i18n.T("placeholder_custom_song")

	return basicInfo
}

// ProgressInfo, only change during download

type PreloadedSongProgressInfo struct {
	Total         int64
	Downloaded    int64
	IsDownloading bool
}

func (ps *StatefulSong) GetProgressInfo() PreloadedSongProgressInfo {
	return PreloadedSongProgressInfo{
		Total:      ps.TotalSize,
		Downloaded: ps.DownloadedSize,

		IsDownloading: ps.sm.DownloadStatus == Downloading,
	}
}

func (ps *StatefulSong) GetError() string {
	if ps.PreloadError != nil {
		return ps.PreloadError.Error()
	}
	return ""
}

// TimeInfo, only change during play

type PreloadedSongTimeInfo struct {
	Progress float64
	Text     string

	IsPlaying   bool
	IsCountdown bool
}

func (ps *StatefulSong) GetTimeInfo() PreloadedSongTimeInfo {
	duration := ps.info.Duration

	if ps.sm.PlayStatus == SyncPlaying {
		if ps.TimePassed < 0 {
			countdown := (-ps.TimePassed).Seconds()
			return PreloadedSongTimeInfo{
				Progress: countdown / 10.0,
				Text: i18n.T("wrapper_countdown", goeasyi18n.Options{
					Data: map[string]any{"Countdown": int(countdown)},
				}),
				IsCountdown: true,
			}
		}
		if duration > 0 {
			return PreloadedSongTimeInfo{
				Progress:  float64(ps.TimePassed.Milliseconds()) / float64(duration.Milliseconds()),
				Text:      fmt.Sprintf("%s / %s", utils.PrettyTime(ps.TimePassed), utils.PrettyTime(duration)),
				IsPlaying: true,
			}
		}

		return PreloadedSongTimeInfo{
			Progress:  0,
			Text:      fmt.Sprintf("%s / ?", utils.PrettyTime(ps.TimePassed)),
			IsPlaying: true,
		}
	}

	if duration > 0 {
		return PreloadedSongTimeInfo{
			Progress:  -1,
			Text:      fmt.Sprintf("? / %s", utils.PrettyTime(duration)),
			IsPlaying: ps.sm.IsPlaying(),
		}
	}
	return PreloadedSongTimeInfo{
		Progress:  -1,
		Text:      "?",
		IsPlaying: ps.sm.IsPlaying(),
	}
}

// StatusInfo, only change during download

type PreloadedSongStatusInfo struct {
	Status string
	Color  fyne.ThemeColorName

	DynamicUntil time.Time
	PreloadError error
}

func (ps *StatefulSong) GetStatusInfo() PreloadedSongStatusInfo {
	var color fyne.ThemeColorName
	switch ps.sm.DownloadStatus {
	case Initial, Removed, NotAvailable, Disabled:
		color = theme.ColorNamePlaceHolder
	case Pending, CoolingDown:
		color = theme.ColorNameWarning
	case Resolving, Requesting, Downloading:
		color = theme.ColorNamePrimary
	case Downloaded:
		color = theme.ColorNameSuccess
	case Failed, Refused:
		color = theme.ColorNameError
	}

	var until time.Time
	switch ps.sm.DownloadStatus {
	case CoolingDown:
		until = ps.sm.task.Traffic.ScheduledTime()
	case Failed:
		until = ps.sm.retryUntil
	default:
	}

	var status string
	if until.IsZero() {
		status = i18n.T(fmt.Sprintf("status_download_%d", ps.sm.DownloadStatus))
	} else {
		status = i18n.T(fmt.Sprintf("status_download_%d", ps.sm.DownloadStatus), goeasyi18n.Options{
			Data: map[string]any{"Seconds": int(time.Until(until).Seconds())},
		})
	}

	return PreloadedSongStatusInfo{
		Status: status,
		Color:  color,

		DynamicUntil: until,
		PreloadError: ps.PreloadError,
	}
}
