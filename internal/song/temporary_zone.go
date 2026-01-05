package song

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type TemporarySong struct {
	key    string
	song   *PreloadedSong
	inList bool

	useCount atomic.Int32

	removeCh chan struct{}
}

func (t *TemporarySong) tryRemove() bool {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if t.useCount.Load() <= 0 {
		if !t.inList {
			delete(temporaryMap, t.key)
			t.song.RemoveFromList()
		}
		return true
	} else {
		return false
	}
}
func (t *TemporarySong) removeAfterTimeout() bool {
	for {
		select {
		case <-t.removeCh:
			continue
		case <-time.After(time.Second):
			return t.tryRemove()
		}
	}
}
func (t *TemporarySong) ReleaseIfCancelled(ctx context.Context) {
	<-ctx.Done()
	if t.useCount.Add(-1) <= 0 {
		t.removeCh <- struct{}{}
	}
}
func (t *TemporarySong) WatchRemove() {
	for {
		<-t.removeCh
		if t.removeAfterTimeout() {
			return
		}
	}
}

func NewTemporarySong(key string, song *PreloadedSong) *TemporarySong {
	t := &TemporarySong{
		key:      key,
		song:     song,
		removeCh: make(chan struct{}),
	}

	return t
}

var temporaryMap = make(map[string]*TemporarySong)
var mapMutex sync.Mutex

func GetTemporaryPyPySong(id int, ctx context.Context) *PreloadedSong {
	return findOrCreateTemporarySong(fmt.Sprintf("pypy_%d", id), ctx, func() *PreloadedSong {
		return CreatePreloadedPyPySong(id)
	})
}
func GetTemporaryWannaSong(id int, ctx context.Context) *PreloadedSong {
	return findOrCreateTemporarySong(fmt.Sprintf("wanna_%d", id), ctx, func() *PreloadedSong {
		return CreatePreloadedWannaSong(id)
	})
}
func GetTemporaryCustomSong(url string, ctx context.Context) *PreloadedSong {
	return findOrCreateTemporarySong(url, ctx, func() *PreloadedSong {
		return CreatePreloadedCustomSong(url)
	})
}

func findOrCreateTemporarySong(key string, ctx context.Context, create func() *PreloadedSong) *PreloadedSong {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	song, ok := temporaryMap[key]
	if !ok {
		song = NewTemporarySong(key, create())
		go song.WatchRemove()
		temporaryMap[key] = song
	}

	song.useCount.Add(1)
	go song.ReleaseIfCancelled(ctx)

	return song.song
}

func GetPyPySongForList(id int) *PreloadedSong {
	song := drawFromTemporary(fmt.Sprintf("pypy_%d", id))
	if song == nil {
		song = CreatePreloadedPyPySong(id)
	}
	return song
}
func GetWannaSongForList(id int) *PreloadedSong {
	song := drawFromTemporary(fmt.Sprintf("wanna_%d", id))
	if song == nil {
		song = CreatePreloadedWannaSong(id)
	}
	return song
}
func GetDuDuSongForList(id int) *PreloadedSong {
	song := drawFromTemporary(fmt.Sprintf("dudu_%d", id))
	if song == nil {
		song = CreatePreloadedDuDuSong(id)
	}
	return song
}
func GetCustomSongForList(url string) *PreloadedSong {
	song := drawFromTemporary(url)
	if song == nil {
		song = CreatePreloadedCustomSong(url)
	}
	return song
}

func drawFromTemporary(key string) *PreloadedSong {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if song, ok := temporaryMap[key]; ok {
		song.inList = true
		delete(temporaryMap, key)
		return song.song
	}

	return nil
}
