package types

type QueueItem struct {
	SongNum    int    `json:"songNum"`
	VideoName  string `json:"videoName"`
	Length     int    `json:"length"`
	URL        string `json:"url"`
	PlayerName string `json:"playerName"`
	Group      string `json:"group"`
}
