package cache

import (
	"io"
	"log"
	"net/http"

	"github.com/wzhqwq/PyPyDancePreloader/internal/song/raw_song"
)

var savedResponse []byte

func loadSongs() error {
	// load songs from https://jd.pypy.moe/api/v2/songs
	log.Println("loading songs")
	resp, err := http.Get("https://jd.pypy.moe/api/v2/songs")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	savedResponse, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return raw_song.ProcessSongList(savedResponse)
}

func GetSongListBytes() []byte {
	if savedResponse == nil {
		err := loadSongs()
		if err != nil {
			log.Println("Failed to load songs:", err)
			return nil
		}
	}
	return savedResponse
}
