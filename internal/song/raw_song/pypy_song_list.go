package raw_song

import (
	"bytes"
	"encoding/json"
)

var pyPySongMap map[int]PyPyDanceSong
var pypyGroups []string

func FindPyPySong(id int) (*PyPyDanceSong, bool) {
	song, ok := pyPySongMap[id]
	return &song, ok
}

type PyPyDanceListResponse struct {
	UpdatedAt int             `json:"updatedAt"`
	Groups    []string        `json:"groups"`
	Songs     []PyPyDanceSong `json:"songs"`
}

func ProcessPyPyDanceList(resp []byte) error {
	var data PyPyDanceListResponse
	err := json.NewDecoder(bytes.NewReader(resp)).Decode(&data)
	if err != nil {
		return err
	}

	pyPySongMap = make(map[int]PyPyDanceSong)
	for _, song := range data.Songs {
		pyPySongMap[song.ID] = song
	}
	pypyGroups = data.Groups

	return nil
}
