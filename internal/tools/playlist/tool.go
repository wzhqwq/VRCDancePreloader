package playlist

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

var logger = utils.NewLogger("Playlist")

func initialize() error {
	currentPlaylist = newPlayList()
	notifyNewList(currentPlaylist)
	return nil
}

func destroy() error {
	if currentPlaylist == nil {
		return nil
	}
	currentPlaylist.StopAll()
	currentPlaylist = nil

	return nil
}

type Tool struct {
	service.ConfigurableTool
}

func New() *Tool {
	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool("playlist", initialize, destroy, logger),
	}
}
