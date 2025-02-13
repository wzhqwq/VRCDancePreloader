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
	title := i18n.T("placeholder_empty_song")
	if ps.RandomPlay {
		title = "random_play"
	}
	return PreloadedSongInfo{
		Title: title,
	}
}

// ProgressInfo, only change during download

type PreloadedSongProgressInfo struct {
	Progress         float64
	DownloadedPretty string
	IsDownloading    bool
}

func (ps *PreloadedSong) GetProgressInfo() PreloadedSongProgressInfo {
	text := i18n.T("placeholder_unknown_size")
	if ps.sm.DownloadStatus == Downloading {
		text = fmt.Sprintf(
			"%s / %s",
			utils.PrettyByteSize(ps.DownloadedSize),
			utils.PrettyByteSize(ps.TotalSize),
		)
	}
	if ps.sm.DownloadStatus == Downloaded {
		text = utils.PrettyByteSize(ps.TotalSize)
	}
	return PreloadedSongProgressInfo{
		Progress:         float64(ps.DownloadedSize) / float64(ps.TotalSize),
		DownloadedPretty: text,
		IsDownloading:    ps.sm.DownloadStatus == Downloading,
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
		Progress:  ps.TimePassed / ps.Duration,
		Text:      text,
		IsPlaying: ps.sm.PlayStatus == Playing,
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
	case Initial, Pending, Removed, NotAvailable:
		color = theme.ColorNamePlaceHolder
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
