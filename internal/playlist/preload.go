package playlist

func (pl *PlayList) CriticalUpdate() {
	select {
	case pl.criticalUpdateCh <- struct{}{}:
	default:
	}
}

func (pl *PlayList) Preload() {
	scanned := 0
	for _, item := range pl.Items {
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
