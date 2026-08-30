package song

type DownloadStatus int

const (
	// Initial is the initial state of every song
	Initial DownloadStatus = iota
	// Pending is when the song is waiting for the download to start,
	// either because previous songs are still downloading
	// or it's queue-jumped by a higher priority song
	Pending
	// CoolingDown is when the song is waiting for the scheduler but not the retry delay
	CoolingDown

	// Resolving is when the final video url is being resolved
	Resolving
	// Requesting is when the final video url is being requested
	Requesting
	// Downloading is when the song is downloading
	Downloading
	// Downloaded is when the song is downloaded to the disk
	Downloaded

	// Failed is when the song failed to download, will be retried
	Failed
	// Removed is when the song is removed from the playlist
	Removed

	// NotAvailable means the song cannot be cached by now
	NotAvailable
	// Disabled means the song is disabled by the user
	Disabled
	// Refused means the request is refused by the server
	Refused
)

type PlayStatus string

const (
	Queued      PlayStatus = "queued"
	Playing     PlayStatus = "playing"
	SyncPlaying PlayStatus = "sync_playing"
	Ended       PlayStatus = "ended"
)

func (sm *StateMachine) IsDownloadLoopStarted() bool {
	return sm.IsDownloadNeeded() && sm.DownloadStatus != Initial
}
func (sm *StateMachine) IsDownloadNeeded() bool {
	return sm.DownloadStatus != Downloaded && sm.DownloadStatus != Removed &&
		sm.DownloadStatus != NotAvailable && sm.DownloadStatus != Disabled && sm.DownloadStatus != Refused
}
func (sm *StateMachine) IsPlaying() bool {
	return sm.PlayStatus == Playing || sm.PlayStatus == SyncPlaying
}
