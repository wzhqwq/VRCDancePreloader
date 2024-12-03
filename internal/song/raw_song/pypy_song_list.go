package raw_song

import (
	"bytes"
	"encoding/json"
	"log"
)

var pypySongMap map[int]PyPyDanceSong
var songGroups []string

func FindSong(id int) (*PyPyDanceSong, bool) {
	song, ok := pypySongMap[id]
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

	pypySongMap = make(map[int]PyPyDanceSong)
	for _, song := range data.Songs {
		pypySongMap[song.ID] = song
	}
	songGroups = data.Groups

	return nil
}
