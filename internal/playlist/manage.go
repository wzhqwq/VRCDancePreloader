package playlist

func Init(maxPreload int) {
	currentPlaylist = newPlayList(maxPreload)
	notifyNewList(currentPlaylist)
	currentPlaylist.Start()
}

func StopPlayList() {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.StopAll()
	currentPlaylist = nil
}

func SetMaxPreload(maxPreload int) {
	if currentPlaylist == nil {
		return
	}
	currentPlaylist.Lock()
	defer currentPlaylist.Unlock()
	currentPlaylist.maxPreload = maxPreload
}
