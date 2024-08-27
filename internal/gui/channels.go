package gui

import (
	"sync"

	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

var updatePlayItemCh chan types.PlayItemI = make(chan types.PlayItemI, 100)
var addPlayItemCh chan []types.PlayItemI = make(chan []types.PlayItemI, 1)
var removePlayItemCh chan []types.PlayItemI = make(chan []types.PlayItemI, 1)

var channelMutationMutex sync.Mutex

func UpdatePlayItem(item types.PlayItemI) {
	// if the channel is full, then take the oldest item out
	channelMutationMutex.Lock()
	select {
	case updatePlayItemCh <- item:
	case <-updatePlayItemCh:
		updatePlayItemCh <- item
	default:
	}
	channelMutationMutex.Unlock()
}

func AddPlayItem(item types.PlayItemI) {
	// if the channel is full, then take out and append
	channelMutationMutex.Lock()
	select {
	case addPlayItemCh <- []types.PlayItemI{item}:
	case items := <-addPlayItemCh:
		items = append(items, item)
		addPlayItemCh <- items
	}
	channelMutationMutex.Unlock()
}

func RemovePlayItem(item types.PlayItemI) {
	// if the channel is full, then take out and append
	channelMutationMutex.Lock()
	select {
	case removePlayItemCh <- []types.PlayItemI{item}:
	default:
		items := <-removePlayItemCh
		items = append(items, item)
		removePlayItemCh <- items
	}
	channelMutationMutex.Unlock()
}
