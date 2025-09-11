package song

type LiveFullInfo struct {
	ID int64 `json:"id"`

	SongID string `json:"songId"`
	Title  string `json:"title"`
	Adder  string `json:"adder"`
	Group  string `json:"group"`

	PlayStatus     string `json:"playStatus"`
	DownloadStatus string `json:"downloadStatus"`

	Duration   int `json:"duration"`
	TimePassed int `json:"timePassed"`

	DownloadProgress float64 `json:"downloadProgress"`

	Error string `json:"error"`
}

func (ps *PreloadedSong) LiveFullInfo() LiveFullInfo {
	basic := ps.GetInfo()
	var err string
	if ps.PreloadError != nil {
		err = ps.PreloadError.Error()
	}
	var progress = float64(0)
	if ps.TotalSize != 0 {
		progress = float64(ps.DownloadedSize) / float64(ps.TotalSize) * 100
	}
	return LiveFullInfo{
		ID: ps.ID,

		SongID: basic.ID,
		Title:  basic.Title,
		Adder:  basic.Adder,
		Group:  basic.Group,

		PlayStatus:     string(ps.sm.PlayStatus),
		DownloadStatus: string(ps.sm.DownloadStatus),

		Duration:   int(ps.Duration.Milliseconds()),
		TimePassed: int(ps.TimePassed.Milliseconds()),

		DownloadProgress: progress,

		Error: err,
	}
}

type LiveStatusChange struct {
	ID int64 `json:"id"`

	DownloadStatus string `json:"downloadStatus"`

	Error string `json:"error"`
}

func (ps *PreloadedSong) LiveStatusChange() LiveStatusChange {
	var err string
	if ps.PreloadError != nil {
		err = ps.PreloadError.Error()
	}
	return LiveStatusChange{
		ID: ps.ID,

		DownloadStatus: string(ps.sm.DownloadStatus),

		Error: err,
	}
}

type LiveProgressChange struct {
	ID int64 `json:"id"`

	DownloadProgress float64 `json:"downloadProgress"`
}

func (ps *PreloadedSong) LiveProgressChange() LiveProgressChange {
	return LiveProgressChange{
		ID: ps.ID,

		DownloadProgress: float64(ps.DownloadedSize) / float64(ps.TotalSize) * 100,
	}
}

type LivePlayStatusChange struct {
	ID int64 `json:"id"`

	TimePassed int    `json:"timePassed"`
	PlayStatus string `json:"playStatus"`
}

func (ps *PreloadedSong) LivePlayStatusChange() LivePlayStatusChange {
	return LivePlayStatusChange{
		ID: ps.ID,

		TimePassed: int(ps.TimePassed.Milliseconds()),
		PlayStatus: string(ps.sm.PlayStatus),
	}
}

type LiveBasicInfoChange struct {
	ID int64 `json:"id"`

	Title    string `json:"title"`
	Group    string `json:"group"`
	Duration int    `json:"duration"`
}

func (ps *PreloadedSong) LiveBasicInfoChange() LiveBasicInfoChange {
	basic := ps.GetInfo()
	return LiveBasicInfoChange{
		ID:       ps.ID,
		Title:    basic.Title,
		Group:    basic.Group,
		Duration: int(ps.Duration.Milliseconds()),
	}
}
