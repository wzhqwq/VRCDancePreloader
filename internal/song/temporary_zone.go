package song

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type TemporarySong struct {
	key    string
	song   *StatefulSong
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
	}

	return false
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

func NewTemporarySong(key string, song *StatefulSong) *TemporarySong {
	t := &TemporarySong{
		key:      key,
		song:     song,
		removeCh: make(chan struct{}),
	}

	return t
}

var temporaryMap = make(map[string]*TemporarySong)
var mapMutex sync.Mutex

func GetTemporarySongByInternalId(id string, ctx context.Context) *StatefulSong {
	return findOrCreateTemporarySong(id, ctx, func() *StatefulSong {
		return CreateStatefulSongByInternalId(id)
	})
}

func findOrCreateTemporarySong(key string, ctx context.Context, create func() *StatefulSong) *StatefulSong {
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

func MakePlaylistSongByInternalId(id string) *StatefulSong {
	song := drawFromTemporary(id)
	if song == nil {
		song = CreateStatefulSongByInternalId(id)
	}
	return song
}

func drawFromTemporary(key string) *StatefulSong {
	mapMutex.Lock()
	defer mapMutex.Unlock()

	if song, ok := temporaryMap[key]; ok {
		song.inList = true
		delete(temporaryMap, key)
		return song.song
	}

	return nil
}
