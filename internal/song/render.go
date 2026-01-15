package song

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/i18n"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// Info, immutable

type PreloadedSongInfo struct {
	ID          string
	Title       string
	Group       string
	Adder       string
	Size        string
	OriginalURL string
}

func (ps *PreloadedSong) GetInfo() PreloadedSongInfo {
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

	basicInfo := PreloadedSongInfo{
		ID: ps.GetSongId(),

		Adder:       adder,
		Size:        size,
		OriginalURL: ps.GetOriginalUrl(),
	}

	if ps.PyPySong != nil {
		if ps.InfoNa {
			basicInfo.Title = fmt.Sprintf("PyPyDance %d", ps.PyPySong.ID)
			basicInfo.Group = ""
		} else {
			basicInfo.Title = ps.PyPySong.Name
			basicInfo.Group = ps.PyPySong.GetGroupName()
		}
		return basicInfo
	}
	if ps.WannaSong != nil {
		if ps.InfoNa {
			basicInfo.Title = fmt.Sprintf("WannaDance %d", ps.WannaSong.DanceId)
			basicInfo.Group = ""
		} else {
			basicInfo.Title = ps.WannaSong.FullTitle()
			basicInfo.Group = ps.WannaSong.Group
		}
		return basicInfo
	}
	if ps.DuDuSong != nil {
		if ps.InfoNa {
			basicInfo.Title = fmt.Sprintf("DuDuFitDance %d", ps.DuDuSong.ID)
			basicInfo.Group = ""
		} else {
			basicInfo.Title = ps.DuDuSong.FullTitle()
			basicInfo.Group = ps.DuDuSong.Group
		}
		return basicInfo
	}
	if ps.CustomSong != nil {
		basicInfo.Title = ps.CustomSong.Name
		basicInfo.Group = i18n.T("placeholder_custom_song")
		return basicInfo
	}

	basicInfo.Title = i18n.T("placeholder_unknown_song")
	basicInfo.Group = i18n.T("placeholder_custom_song")
	return basicInfo
}

// ProgressInfo, only change during download

type PreloadedSongProgressInfo struct {
	Total         int64
	Downloaded    int64
	IsDownloading bool
}

func (ps *PreloadedSong) GetProgressInfo() PreloadedSongProgressInfo {
	return PreloadedSongProgressInfo{
		Total:      ps.TotalSize,
		Downloaded: ps.DownloadedSize,

		IsDownloading: ps.sm.DownloadStatus == Downloading,
	}
}

func (ps *PreloadedSong) GetError() string {
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

func (ps *PreloadedSong) GetTimeInfo() PreloadedSongTimeInfo {
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
		return PreloadedSongTimeInfo{
			Progress:  float64(ps.TimePassed.Milliseconds()) / float64(ps.Duration.Milliseconds()),
			Text:      fmt.Sprintf("%s / %s", utils.PrettyTime(ps.TimePassed), utils.PrettyTime(ps.Duration)),
			IsPlaying: true,
		}
	}
	return PreloadedSongTimeInfo{
		Progress:  -1,
		Text:      utils.PrettyTime(ps.Duration),
		IsPlaying: ps.sm.IsPlaying(),
	}
}

// StatusInfo, only change during download

type PreloadedSongStatusInfo struct {
	Status string
	Color  fyne.ThemeColorName

	PreloadError error
}

func (ps *PreloadedSong) GetStatusInfo() PreloadedSongStatusInfo {
	var color fyne.ThemeColorName
	switch ps.sm.DownloadStatus {
	case Initial, Removed, NotAvailable, Disabled:
		color = theme.ColorNamePlaceHolder
	case Pending, CoolingDown:
		color = theme.ColorNameWarning
	case Requesting, Downloading:
		color = theme.ColorNamePrimary
	case Downloaded:
		color = theme.ColorNameSuccess
	case Failed:
		color = theme.ColorNameError
	}
	return PreloadedSongStatusInfo{
		Status: i18n.T(fmt.Sprintf("status_%s", ps.sm.DownloadStatus)),
		Color:  color,

		PreloadError: ps.PreloadError,
	}
}
