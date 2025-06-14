package playlist

func (pl *PlayList) CriticalUpdate() {
	select {
	case pl.criticalUpdateCh <- struct{}{}:
	default:
	}
}

func (pl *PlayList) Preload() {
	scanned := 0
	items := pl.GetItemsSnapshot()
	for _, item := range items {
		if scanned > pl.maxPreload {
			break
		}
		item.PreloadSong()
		if scanned <= 1 {
			item.PrioritizeSong()
		}
		scanned++
	}
}
