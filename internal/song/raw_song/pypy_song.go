package raw_song

type PyPyDanceSong struct {
	ID          int      `json:"i"`
	Group       int      `json:"g"`
	Name        string   `json:"n"`
	End         int      `json:"e"`
	OriginalURL []string `json:"o"`
	// Tags        []string `json:"t"`
}

func (song *PyPyDanceSong) GetGroupName() string {
	return pypyGroups[song.Group]
}
