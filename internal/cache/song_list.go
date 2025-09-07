package cache

import (
	"io"
	"log"
	"net/http"

	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

func loadSongs() error {
	var savedResponse []byte

	log.Println("loading PyPyDance songs")
	resp, err := http.Get("https://api.pypy.dance/bundle")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	savedResponse, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = raw_song.ProcessPyPyDanceList(savedResponse)
	if err != nil {
		return err
	}

	log.Println("loading WannaDance songs")
	resp, err = http.Get("https://api.udon.dance/Api/Songs/list")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	savedResponse, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = raw_song.ProcessWannaDanceList(savedResponse)
	if err != nil {
		return err
	}

	return nil
}
