package thumbnails

import (
	"bytes"
	"embed"
	"image"
	"image/jpeg"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/nfnt/resize"
	"github.com/stephennancekivell/go-future/future"
	"github.com/wzhqwq/VRCDancePreloader/internal/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

//go:embed *.jpg
var thumbnailFS embed.FS

var groupNameToThumbnail = map[string]string{
	// shared
	"Just Dance Solo":     "thumbnail-justdance-solo.jpg",
	"Just Dance Duet":     "thumbnail-justdance-duet.jpg",
	"Just Dance Trio":     "thumbnail-justdance-trio.jpg",
	"Just Dance Crew":     "thumbnail-justdance-crew.jpg",
	"FitDance":            "thumbnail-fitdance.jpg",
	"Fitness Marshall":    "thumbnail-marshall.jpg",
	"Mylee Dance":         "thumbnail-mylee.jpg",
	"TML Crew":            "thumbnail-tml.jpg",
	"Golfy Dance Fitness": "thumbnail-golfy.jpg",
	"SouthVibes":          "thumbnail-southvibes.jpg",

	// PyPyDance only
	"Just Dance":     "thumbnail-justdance-solo.jpg",
	"Others (K-POP)": "thumbnail-kpop.jpg",
	"Others (J-POP)": "thumbnail-jpop.jpg",

	// WannaDance only
	"Song^_^":       "thumbnail-song.jpg",
	"Fol2esTz":      "thumbnail-fol2estz.jpg",
	"Lisa Rhee":     "thumbnail-lisa.jpg",
	"足太ぺんた":         "thumbnail-penta.jpg",
	"Other Fitness": "thumbnail-fitness.jpg",
	"Other K-POP":   "thumbnail-kpop.jpg",
}

var defaultThumbnail = "thumbnail-default.jpg"

var thumbnails = map[string]image.Image{}
var thumbnailsMutex sync.RWMutex

func getThumbnail(name string) image.Image {
	thumbnailsMutex.RLock()
	if r, ok := thumbnails[name]; ok {
		thumbnailsMutex.RUnlock()
		return r
	}
	thumbnailsMutex.RUnlock()
	return loadThumbnail(name)
}

func loadThumbnail(name string) image.Image {
	thumbnailsMutex.Lock()
	defer thumbnailsMutex.Unlock()

	f, err := thumbnailFS.Open(name)
	if err != nil {
		return nil
	}
	defer f.Close()

	img, err := jpeg.Decode(f)
	if err != nil {
		return nil
	}

	thumbnails[name] = img

	return img
}

func GetGroupThumbnail(groupName string) image.Image {
	if thumbnail, ok := groupNameToThumbnail[groupName]; ok {
		return getThumbnail(thumbnail)
	}
	return getThumbnail(defaultThumbnail)
}

func GetDefaultThumbnail() image.Image {
	return getThumbnail(defaultThumbnail)
}

type AsyncImage struct {
	i      future.Future[image.Image]
	loaded bool
}

var cache = utils.NewWeakCache[AsyncImage](100)

func GetThumbnailImage(id, url string) image.Image {
	if group, ok := strings.CutPrefix(url, "group:"); ok {
		i := GetGroupThumbnail(group)
		return i
	}

	key := url
	if id != "" {
		key = id
	}
	if i, ok := cache.Get(key); ok {
		return i.i.Get()
	}

	if url == "" {
		return getThumbnail(defaultThumbnail)
	}

	i := future.New(func() image.Image {
		defer func() {
			if i, ok := cache.Get(key); ok {
				i.loaded = true
			}
		}()

		log.Println("Downloading thumbnail from ", url)
		resp, err := requesting.RequestThumbnail(url)
		if err != nil {
			log.Println("Failed to get thumbnail:", err)
			return getThumbnail(defaultThumbnail)
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("Unable to read image data", err)
			return getThumbnail(defaultThumbnail)
		}

		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			log.Println("Failed to decode image:", err)
			return getThumbnail(defaultThumbnail)
		}

		return resize.Resize(320, 0, img, resize.Bilinear)
	})

	cache.Set(key, AsyncImage{i: i, loaded: false})

	return i.Get()
}

func HasThumbnailCachedAndLoaded(key string) bool {
	f, ok := cache.Get(key)
	if ok {
		return f.loaded
	}
	return false
}
