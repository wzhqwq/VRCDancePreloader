package song

type DownloadStatus string

const (
	// Initial is the initial state of every song
	Initial DownloadStatus = "initial"
	// Pending is when the song is waiting for the download to start,
	// either because previous songs are still downloading
	// or it's queue-jumped by a higher priority song
	Pending DownloadStatus = "pending"
	// CoolingDown is when the song is waiting for the scheduler but not the retry delay
	CoolingDown DownloadStatus = "cooling_down"

	// Requesting is when the song is requesting the download
	Requesting DownloadStatus = "requesting"
	// Downloading is when the song is downloading
	Downloading DownloadStatus = "downloading"
	// Downloaded is when the song is downloaded to the disk
	Downloaded DownloadStatus = "downloaded"

	// Failed is when the song failed to download, will be retried
	Failed DownloadStatus = "failed"
	// Removed is when the song is removed from the playlist
	Removed DownloadStatus = "removed"

	// NotAvailable means the song cannot be cached by now
	NotAvailable DownloadStatus = "na"
	// Disabled means the song is disabled by the user
	Disabled DownloadStatus = "disabled"
)

type PlayStatus string

const (
	Queued  PlayStatus = "queued"
	Playing PlayStatus = "playing"
	Ended   PlayStatus = "ended"
)

func (sm *StateMachine) IsDownloadLoopStarted() bool {
	return sm.DownloadStatus == Pending || sm.DownloadStatus == CoolingDown || sm.DownloadStatus == Requesting || sm.DownloadStatus == Downloading
}
func (sm *StateMachine) IsDownloadNeeded() bool {
	return sm.DownloadStatus != Downloaded && sm.DownloadStatus != Removed && sm.DownloadStatus != NotAvailable
}
func (sm *StateMachine) CanPreload() bool {
	return sm.DownloadStatus != NotAvailable && (sm.DownloadStatus == Initial || sm.DownloadStatus == Failed)
}
func (sm *StateMachine) IsPlayingLoopStarted() bool {
	return sm.PlayStatus == Playing
}
