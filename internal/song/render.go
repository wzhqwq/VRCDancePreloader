package song

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

// Info, immutable

type PreloadedSongInfo struct {
	ID           string
	Title        string
	Group        string
	Adder        string
	Size         string
	OriginalURL  string
	ThumbnailURL string
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

	if ps.PyPySong != nil {
		return PreloadedSongInfo{
			ID:           ps.GetId(),
			Title:        ps.PyPySong.Name,
			Group:        ps.PyPySong.GetGroupName(),
			Adder:        adder,
			Size:         size,
			OriginalURL:  ps.GetOriginalUrl(),
			ThumbnailURL: ps.GetThumbnailUrl(),
		}
	}
	if ps.CustomSong != nil {
		return PreloadedSongInfo{
			ID:           ps.GetId(),
			Title:        ps.CustomSong.Name,
			Group:        i18n.T("placeholder_custom_song"),
			Adder:        adder,
			Size:         size,
			OriginalURL:  ps.GetOriginalUrl(),
			ThumbnailURL: ps.GetThumbnailUrl(),
		}
	}
	return PreloadedSongInfo{}
}

// ProgressInfo, only change during download

type PreloadedSongProgressInfo struct {
	Progress        float64
	DownloadedBytes string
}

func (ps *PreloadedSong) GetProgressInfo() PreloadedSongProgressInfo {
	return PreloadedSongProgressInfo{
		Progress:        float64(ps.DownloadedSize) / float64(ps.TotalSize),
		DownloadedBytes: utils.PrettyByteSize(ps.DownloadedSize),
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
	Progress  float64
	Text      string
	IsPlaying bool
}

func (ps *PreloadedSong) GetTimeInfo() PreloadedSongTimeInfo {
	text := utils.PrettyTime(ps.Duration)
	if ps.sm.PlayStatus == Playing {
		text = fmt.Sprintf("%s / %s", utils.PrettyTime(ps.TimePassed), text)
	}
	return PreloadedSongTimeInfo{
		Progress:  float64(ps.TimePassed) / float64(ps.Duration),
		Text:      text,
		IsPlaying: ps.sm.PlayStatus == Playing,
	}
}

// StatusInfo, only change during download

type PreloadedSongStatusInfo struct {
	Status string
	Color  fyne.ThemeColorName
}

func (ps *PreloadedSong) GetStatusInfo() PreloadedSongStatusInfo {
	var status string
	var color fyne.ThemeColorName
	switch ps.sm.DownloadStatus {
	case Initial:
		status = i18n.T("status_initial")
		color = theme.ColorNamePlaceHolder
	case Pending:
		status = i18n.T("status_pending")
		color = theme.ColorNamePlaceHolder
	case Requesting:
		status = i18n.T("status_requesting")
		color = theme.ColorNamePrimary
	case Downloading:
		status = i18n.T("status_downloading")
		color = theme.ColorNamePrimary
	case Downloaded:
		status = i18n.T("status_downloaded")
		color = theme.ColorNameSuccess
	case Failed:
		status = i18n.T("status_failed")
		color = theme.ColorNameError
	}
	return PreloadedSongStatusInfo{
		Status: status,
		Color:  color,
	}
}
