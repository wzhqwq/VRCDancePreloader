package types

type Song struct {
	ID          int      `json:"id"`
	Group       int      `json:"group"`
	Volume      float64  `json:"volume"`
	Name        string   `json:"name"`
	Flip        bool     `json:"flip"`
	Start       int      `json:"start"`
	End         int      `json:"end"`
	SkipRandom  bool     `json:"skipRandom"`
	OriginalURL []string `json:"originalUrl"`
}
