package raw_song

import (
	"bytes"
	"encoding/json"
	"log"
)

var pyPySongMap map[int]PyPyDanceSong
var songGroups []string

func FindPyPySong(id int) (*PyPyDanceSong, bool) {
	song, ok := pyPySongMap[id]
	return &song, ok
}

type SongsResponse struct {
	UpdatedAt int             `json:"updatedAt"`
	Groups    []string        `json:"groups"`
	Songs     []PyPyDanceSong `json:"songs"`
}

func ProcessSongList(resp []byte) error {
	var data SongsResponse
	err := json.NewDecoder(bytes.NewReader(resp)).Decode(&data)
	if err != nil {
		return err
	}

	log.Printf("loaded %d songs\n", len(data.Songs))

	pyPySongMap = make(map[int]PyPyDanceSong)
	for _, song := range data.Songs {
		pyPySongMap[song.ID] = song
	}
	songGroups = data.Groups

	return nil
}
