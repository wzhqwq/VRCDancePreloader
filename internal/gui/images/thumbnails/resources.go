package thumbnails

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"sync"

	"github.com/nfnt/resize"
	"github.com/wzhqwq/VRCDancePreloader/internal/tools/requesting"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"golang.org/x/sync/semaphore"
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
	"Just Dance":     "thumbnail-justdance.jpg",
	"Others (K-POP)": "thumbnail-kpop.jpg",
	"Others (J-POP)": "thumbnail-jpop.jpg",

	// WannaDance only
	"Just Dance Fan Made": "thumbnail-justdance-fan.jpg",
	"Song^_^":             "thumbnail-song.jpg",
	"Fol2esTz":            "thumbnail-fol2estz.jpg",
	"Lisa Rhee":           "thumbnail-lisa.jpg",
	"足太ぺんた":               "thumbnail-penta.jpg",
	"Other Fitness":       "thumbnail-fitness.jpg",
	"Other K-POP":         "thumbnail-kpop.jpg",
	"Michael Jackson":     "thumbnail-michael.jpg",

	// DuDuFitDance only
	"DuDu FitDance":      "thumbnail-dudu.jpg",
	"Just Dance Series":  "thumbnail-justdance.jpg",
	"Other FitDance":     "thumbnail-fitness.jpg",
	"Just Dance Fanmade": "thumbnail-justdance-fan.jpg",

	"Michael Jackson: The Experience": "thumbnail-michael.jpg",
}

var defaultThumbnail = "thumbnail-default.jpg"

var thumbnails = map[string]image.Image{}
var thumbnailsMutex sync.RWMutex

var logger = utils.NewLogger("Thumbnails")

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

var thumbnailRequestSem = semaphore.NewWeighted(6)

func GetThumbnailImage(client *requesting.ClientProvider, url string, ctx context.Context) (image.Image, error) {
	err := thumbnailRequestSem.Acquire(context.Background(), 1)
	if err != nil {
		return nil, err
	}
	defer thumbnailRequestSem.Release(1)

	logger.InfoLn("Downloading thumbnail from ", url)
	req, err := client.NewGetRequest(url, ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download thumbnail from %s: %w", url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download thumbnail from %s: %w", url, err)
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode thumbnail from %s: %w", url, err)
	}

	return resize.Resize(320, 0, img, resize.Bilinear), nil
}
