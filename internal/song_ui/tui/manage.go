package tui

import (
	"github.com/wzhqwq/PyPyDancePreloader/internal/playlist"
)

var currentTui *PlayListTui
var stopCh chan struct{}

func Start() {
	ch := playlist.SubscribeNewListEvent()
	stopCh = make(chan struct{})
	go func() {
		for {
			select {
			case <-stopCh:
				if currentTui != nil {
					currentTui.StopCh <- struct{}{}
				}
				return
			case pl := <-ch:
				if currentTui != nil {
					currentTui.StopCh <- struct{}{}
				}
				currentTui = NewPlayListTui(pl)
				go currentTui.RenderLoop()
			}
		}
	}()
}
func Stop() {
	stopCh <- struct{}{}
}
