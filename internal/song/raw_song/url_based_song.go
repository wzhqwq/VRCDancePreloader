package raw_song

import (
	"strconv"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/internal_id"
)

type UrlBasedSongRegistry struct {
	mu       sync.Mutex
	idsByUrl map[string]int
	urls     []string
}

var urlBasedSongs = &UrlBasedSongRegistry{
	idsByUrl: make(map[string]int),
}

func (r *UrlBasedSongRegistry) addUrl(url string) int {
	r.urls = append(r.urls, url)
	r.idsByUrl[url] = len(r.urls) - 1
	return len(r.urls) - 1
}

func (r *UrlBasedSongRegistry) GetInternalIdByUrl(url string) string {
	if id, ok := internal_id.GetIdByUrl(url); ok {
		return id
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.idsByUrl[url]
	if ok {
		return internal_id.UrlBasedInternalPrefix + strconv.Itoa(id)
	}

	return internal_id.UrlBasedInternalPrefix + strconv.Itoa(r.addUrl(url))
}

func (r *UrlBasedSongRegistry) GetUrlByInternalId(internalId string) string {
	if url, ok := internal_id.GetUrlById(internalId); ok {
		return url
	}

	if id, ok := internal_id.CheckIdIsUrlBased(internalId); ok {
		r.mu.Lock()
		defer r.mu.Unlock()

		if id < 0 || id >= len(r.urls) {
			return ""
		}

		return r.urls[id]
	}

	return ""
}

func GetInternalIdByUrl(url string) string {
	return urlBasedSongs.GetInternalIdByUrl(url)
}

func GetUrlByInternalId(internalId string) string {
	return urlBasedSongs.GetInternalIdByUrl(internalId)
}
