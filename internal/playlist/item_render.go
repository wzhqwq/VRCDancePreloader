package playlist

import (
	"fyne.io/fyne/v2/theme"
	"github.com/eduardolat/goeasyi18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/constants"
	"github.com/wzhqwq/PyPyDancePreloader/internal/i18n"
	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
	"github.com/wzhqwq/PyPyDancePreloader/internal/utils"
)

func (i *PlayItem) Render() *types.PlayItemRendered {
	i.updateMutex.Lock()
	defer i.updateMutex.Unlock()

	errorText := ""
	if i.PreloadStatus == constants.Failed && i.Error != nil {
		errorText = i.Error.Error()
	}

	sizeText := i18n.T("placeholder_unknown_size")
	if i.Size > 0 {
		sizeText = utils.PrettyByteSize(i.Size)
	}
	if i.PreloadStatus == constants.Downloading {
		sizeText = utils.PrettyByteSize(i.DownloadedBytes) + "/" + sizeText
	}

	color := theme.ColorNamePlaceHolder
	statusText := i18n.T("status_" + string(i.PreloadStatus))
	switch i.PreloadStatus {
	case constants.Requesting, constants.Downloading:
		color = theme.ColorNamePrimary
	case constants.Downloaded:
		color = theme.ColorNameSuccess
	case constants.Failed:
		color = theme.ColorNameError
	}

	if i.PlayStatus != constants.Pending {
		statusText = statusText + " (" + i18n.T("status_"+string(i.PlayStatus)) + ")"
	}
	switch i.PlayStatus {
	case constants.Playing:
		color = theme.ColorNamePrimary
	case constants.Ended:
		color = theme.ColorNamePlaceHolder
	}

	adder := i18n.T("wrapper_adder", goeasyi18n.Options{
		Data: map[string]any{"Adder": i.Adder},
	})
	if i.Adder == "" {
		adder = i18n.T("placeholder_unknown_adder")
	}
	if i.Adder == "Random" {
		adder = i18n.T("placeholder_random_play")
	}

	i.dirty = false
	return &types.PlayItemRendered{
		ID:    i.ID,
		Title: i.Title,
		Group: i.Group,
		Adder: adder,

		Status:      statusText,
		StatusColor: color,
		Size:        sizeText,
		Index:       i.Index,

		DownloadProgress: i.Progress,
		PlayProgress:     i.Now / float64(i.Duration),
		PlayTimeText:     utils.PrettyTime(i.Now) + "/" + utils.PrettyTime(float64(i.Duration)),
		ErrorText:        errorText,

		IsDownloading: i.PreloadStatus == constants.Downloading,
		IsPlaying:     i.PlayStatus == constants.Playing,
	}
}
func (i *PlayItem) GetInfo() *types.PlayItemInfo {
	return &types.PlayItemInfo{
		ID:    i.ID,
		Title: i.Title,
		Group: i.Group,
		Adder: i.Adder,
		Size:  i.Size,
		URL:   i.URL,
	}
}
