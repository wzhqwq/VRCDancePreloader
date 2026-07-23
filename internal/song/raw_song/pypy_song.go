package raw_song

type PyPyDanceSong struct {
	ID          int      `json:"i"`
	Group       int      `json:"g"`
	Name        string   `json:"n"`
	End         int      `json:"e"`
	OriginalURL []string `json:"o"`
	// Tags        []string `json:"t"`

	GroupName string
}

func (song *PyPyDanceSong) Complete(name, groupName string, end int) {
	song.Name = name
	song.GroupName = groupName
	song.End = end
}
