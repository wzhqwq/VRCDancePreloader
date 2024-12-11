package playlist

func StopPlayList() {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.StopAll()
	currentPlaylist = nil
}
