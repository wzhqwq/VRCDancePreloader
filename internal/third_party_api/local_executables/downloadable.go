package local_executables

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/download"
	"github.com/wzhqwq/VRCDancePreloader/internal/third_party_api/api"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type DownloadableChange string

const (
	BinProgress DownloadableChange = "bin_progress"
	BinVersion  DownloadableChange = "bin_version"
	BinState    DownloadableChange = "bin_state"
)

type DownloadableState string

const (
	BinCheckingLocal   DownloadableState = "bin_checking_local"
	BinInitial         DownloadableState = "bin_initial"
	BinCheckingUpdates DownloadableState = "bin_checking_updates"
	BinUpdateAvailable DownloadableState = "bin_update_available"
	BinIllegal         DownloadableState = "bin_illegal"
	BinDownloading     DownloadableState = "bin_downloading"
	BinDownloaded      DownloadableState = "bin_downloaded"
)

type DownloadableBinary struct {
	Name string
	Path string

	State DownloadableState
	Error error

	Info    BinaryInfo
	Release *api.BriefRelease
	Task    *download.Task

	em *utils.EventManager[DownloadableChange]

	mutex sync.RWMutex

	lowLevel atomic.Bool

	stopCh chan struct{}
}

func NewDownloadableBinary(name string) *DownloadableBinary {
	return &DownloadableBinary{
		Name: name,

		State: BinCheckingLocal,

		em: utils.NewEventManager[DownloadableChange](),

		stopCh: make(chan struct{}),
	}
}

func (d *DownloadableBinary) generateContext(dur time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	go func() {
		select {
		case <-d.stopCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func (d *DownloadableBinary) Valid() bool {
	if d.Path == "" {
		return false
	}
	if _, err := os.Stat(d.Path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (d *DownloadableBinary) Init() {
	if d.State == BinCheckingUpdates || d.State == BinDownloading {
		return
	}

	ctx, cancel := d.generateContext(3 * time.Second)
	defer cancel()

	switch d.Name {
	case "ytdlp":
		d.Info = GetLocalYtDlpInfo(ctx)
	case "deno":
		d.Info = GetLocalDenoInfo(ctx)
	default:
		d.Error = fmt.Errorf("unknown downloadable %s", d.Name)
		d.setState(BinIllegal)
		return
	}

	logger.InfoLn("Local version of", d.Name, "is", d.Info.Version)

	d.setState(BinInitial)
	d.em.NotifySubscribers(BinVersion)
}

func (d *DownloadableBinary) setState(state DownloadableState) {
	d.State = state
	d.em.NotifySubscribers(BinState)
}

func (d *DownloadableBinary) CheckUpdates() {
	if d.State == BinCheckingUpdates {
		return
	}

	d.Error = nil
	d.setState(BinCheckingUpdates)

	ctx, cancel := d.generateContext(10 * time.Second)
	defer cancel()

	var err error
	switch d.Name {
	case "ytdlp":
		d.Release, err = GetLatestYtDlp(ctx, YtDlpStable)
	case "deno":
		d.Release, err = GetLatestDeno(ctx)
	default:
		d.Error = fmt.Errorf("unknown downloadable %s", d.Name)
		d.setState(BinIllegal)
		return
	}

	if err != nil {
		d.Error = err
	}
	if d.Release == nil {
		d.setState(BinInitial)
		return
	}

	logger.InfoLn("New version of", d.Name, "is available:", d.Release.Version)

	d.setState(BinUpdateAvailable)
}

func (d *DownloadableBinary) Upgrade() {
	if d.Release == nil || d.State == BinDownloading {
		return
	}

	d.Error = nil
	d.setState(BinDownloading)

	err := d.DownloadAndReplace()

	if err != nil {
		if !errors.Is(err, context.Canceled) {
			d.Error = err
		}
		d.setState(BinUpdateAvailable)
		return
	}

	logger.InfoLn("Downloaded latest version of", d.Name)

	d.setState(BinCheckingLocal)
	d.checkIntegrityLevel()
	d.Init()
}

func (d *DownloadableBinary) CancelDownload() {
	if d.State == BinDownloading && d.Task != nil {
		d.Task.Cancel()
	}
}
func (d *DownloadableBinary) Stop() {
	d.CancelDownload()
	close(d.stopCh)
}

func (d *DownloadableBinary) Remove() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	err := os.Remove(d.Path)
	if err != nil {
		d.Error = err
		logger.InfoLn("Failed to remove", d.Path)
		d.em.NotifySubscribers(BinState)
	}
	d.Init()
}

func (d *DownloadableBinary) Subscribe() *utils.EventSubscriber[DownloadableChange] {
	return d.em.SubscribeEvent()
}

func (d *DownloadableBinary) HasUpdates() bool {
	return d.Release != nil && d.State != BinCheckingLocal && d.Release.Version != d.Info.Version
}

func (d *DownloadableBinary) UpdateText() string {
	if d.HasUpdates() {
		if d.Info.Exists {
			return "→ " + d.Release.Version
		}

		return "↓ " + d.Release.Version
	}

	return ""
}
