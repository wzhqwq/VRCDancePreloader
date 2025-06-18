package raw_song

type WannaDanceSong struct {
	ID      int    `json:"id"`
	DanceId int    `json:"danceid"`
	Name    string `json:"name"`
	Artist  string `json:"artist"`
	Dancer  string `json:"dancer"`
	Start   int    `json:"start"`
	End     int    `json:"end"`

	Group string
}

func (s WannaDanceSong) FullTitle() string {
	return s.Name + " - " + s.Artist + " | " + s.Dancer
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
