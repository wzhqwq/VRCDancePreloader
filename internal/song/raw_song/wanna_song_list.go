package raw_song

import (
	"bytes"
	"encoding/json"
)

var wannaSongMap map[int]WannaDanceSong

func FindWannaSong(id int) (*WannaDanceSong, bool) {
	song, ok := wannaSongMap[id]
	return &song, ok
}

type WannaDanceListResponse struct {
	Time   string           `json:"time"`
	Groups WannaDanceGroups `json:"groups"`
}
type WannaDanceGroups struct {
	Contents []WannaDanceGroupContent `json:"contents"`
}
type WannaDanceGroupContent struct {
	Name  string           `json:"groupName"`
	Major string           `json:"major"`
	Songs []WannaDanceSong `json:"songInfos"`
}

func ProcessWannaDanceList(resp []byte) error {
	var data WannaDanceListResponse
	err := json.NewDecoder(bytes.NewReader(resp)).Decode(&data)
	if err != nil {
		return err
	}

	wannaSongMap = make(map[int]WannaDanceSong)

	for _, group := range data.Groups.Contents {
		name := group.Name
		for _, song := range group.Songs {
			song.Group = name
			wannaSongMap[song.DanceId] = song
		}
	}

	return nil
}
