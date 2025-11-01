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

func (song *PyPyDanceSong) GetGroupName() string {
	if song.GroupName != "" {
		return song.GroupName
	}
	if song.Group >= len(pypyGroups) {
		return "Unknown"
	}
	return pypyGroups[song.Group]
}

func (song *PyPyDanceSong) Complete(name, groupName string, end int) {
	song.Name = name
	song.GroupName = groupName
	song.End = end
}
