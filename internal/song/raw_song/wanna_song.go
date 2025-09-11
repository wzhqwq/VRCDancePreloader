package raw_song

import "log"

type WannaDanceSong struct {
	ID      int    `json:"id"`
	DanceId int    `json:"danceid"`
	Name    string `json:"name"`
	Artist  string `json:"artist"`
	Dancer  string `json:"dancer"`
	Start   int    `json:"start"`
	End     int    `json:"end"`

	Group string
	Title string
}

func (s *WannaDanceSong) FullTitle() string {
	if s.Title == "" {
		s.Title = s.Name + " - " + s.Artist + " | " + s.Dancer
	}
	log.Println("get title", s.ID, s.Title)
	return s.Title
}

func (s *WannaDanceSong) Complete(title, group string, end int) {
	log.Println("complete", s.ID)
	s.Title = title
	s.Group = group
	s.End = end
}

// full
//{
//	ID            int         `json:"id"`
//	DanceId       int         `json:"danceid"`
//	Name          string      `json:"name"`
//	Artist        string      `json:"artist"`
//	Dancer        string      `json:"dancer"`
//	PlayerCount   int         `json:"playerCount"`
//	Volume        int         `json:"volume"`
//	Start         int         `json:"start"`
//	End           int         `json:"end"`
//	Flip          bool        `json:"flip"`
//	Tag           interface{} `json:"tag"`
//	DoubleWidth   bool        `json:"double_width"`
//	SkipRandom    int         `json:"skip_random"`
//	DisablePublic int         `json:"disable_public"`
//	ShaderMotion  string      `json:"shader_motion"`
//	Rpe           int         `json:"rpe"`
//}
