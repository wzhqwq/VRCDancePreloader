package tui

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/playlist"
)

var currentTui *PlayListTui
var stopCh chan struct{}

func Start() {
	stopCh = make(chan struct{})
	go func() {
		listUpdate := playlist.SubscribeNewListEvent()
		defer listUpdate.Close()

		for {
			select {
			case <-stopCh:
				if currentTui != nil {
					close(currentTui.StopCh)
				}
				return
			case pl := <-listUpdate.Channel:
				if currentTui != nil {
					close(currentTui.StopCh)
				}
				currentTui = NewPlayListTui(pl)
				go currentTui.RenderLoop()
			}
		}
	}()
}
func Stop() {
	close(stopCh)
}
