package raw_song

type PyPyDanceSong struct {
	ID          int      `json:"id"`
	Group       int      `json:"group"`
	Name        string   `json:"name"`
	Start       int      `json:"start"`
	End         int      `json:"end"`
	OriginalURL []string `json:"originalUrl"`
}

// full
// {
//	ID          int      `json:"id"`
//	Group       int      `json:"group"`
//	Volume      float64  `json:"volume"`
//	Name        string   `json:"name"`
//	Flip        bool     `json:"flip"`
//	Start       int      `json:"start"`
//	End         int      `json:"end"`
//	SkipRandom  bool     `json:"skipRandom"`
//	OriginalURL []string `json:"originalUrl"`
//}

func (song *PyPyDanceSong) GetGroupName() string {
	return pypyGroups[song.Group]
}
