package raw_song

type DuDuFitDanceSong struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Dancer string `json:"dancer"`
	Start  int    `json:"start"`
	End    int    `json:"end"`

	//Volume         float64 `json:"volume"`
	//HorizontalFlip bool    `json:"hflip"`
	//PublishedAt    int     `json:"original_published_at"`

	Group string
	Title string
}

func (s *DuDuFitDanceSong) FullTitle() string {
	if s.Title == "" {
		s.Title = s.Name + " - " + s.Artist + " | " + s.Dancer
	}
	return s.Title
}

func (s *DuDuFitDanceSong) Complete(title, group string, end int) {
	s.Title = title
	s.Group = group
	s.End = end
}
