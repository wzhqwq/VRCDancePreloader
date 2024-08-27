package playlist

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/wzhqwq/PyPyDancePreloader/internal/types"
)

var songMap map[int]types.Song
var songGroups []string
var savedResponse []byte
var songLoaded chan struct{} = make(chan struct{}, 1)

func FindSong(id int) (*types.Song, bool) {
	song, ok := songMap[id]
	return &song, ok
}

// SongsResponse: {"updatedAt":1723639203299,"songs":[]}
type SongsResponse struct {
	UpdatedAt int          `json:"updatedAt"`
	Groups    []string     `json:"groups"`
	Songs     []types.Song `json:"songs"`
}

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
	songLoaded <- struct{}{}

	var data SongsResponse
	err = json.NewDecoder(bytes.NewReader(savedResponse)).Decode(&data)
	if err != nil {
		return err
	}

	log.Printf("loaded %d songs\n", len(data.Songs))

	songMap = make(map[int]types.Song)
	for _, song := range data.Songs {
		songMap[song.ID] = song
	}
	songGroups = data.Groups

	return nil
}
