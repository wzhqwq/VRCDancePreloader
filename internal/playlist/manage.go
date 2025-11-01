package playlist

import "github.com/wzhqwq/VRCDancePreloader/internal/stability"

func Init(maxPreload int) {
	currentPlaylist = newPlayList(maxPreload)
	notifyNewList(currentPlaylist)
}

func StopPlayList() {
	if currentPlaylist == nil {
		return
	}
	cancel := stability.PanicIfTimeout("playlist_StopPlaylist")
	defer cancel()
	currentPlaylist.StopAll()
	currentPlaylist = nil
}

func SetMaxPreload(maxPreload int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.maxPreload = maxPreload
	currentPlaylist.CriticalUpdate()
}

func SetEnabledPlatforms(sites []string) {
	// TODO
}

func SetEnabledRooms(rooms []string) {
	// TODO
}
