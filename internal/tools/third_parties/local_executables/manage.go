package local_executables

import (
	"path/filepath"
)

var downloadableMap = map[string]*DownloadableBinary{
	"ytdlp": NewDownloadableBinary("ytdlp"),
	"deno":  NewDownloadableBinary("deno"),
}

func InitYtDlp() {
	b := downloadableMap["ytdlp"]
	if cfg.YtDlpPath == "<vrcdp>" {
		b.SetPathAndCheck(filepath.Join(getLocalBinariesPath(), ytDlpLocalName))

		// Read the setting here, not inside the goroutine: UpdateConfig replaces
		// the whole cfg struct from the config thread, and a read from another
		// goroutine would race with it.
		checkOnStart := cfg.CheckUpdateOnStart

		go func() {
			b.Init()
			if checkOnStart {
				b.CheckUpdates()
			}
		}()
	} else {
		b.CancelDownload()
		b.SetPathAndCheck(cfg.YtDlpPath)
	}
}
func InitDeno() {
	b := downloadableMap["deno"]
	if cfg.DenoPath == "<vrcdp>" {
		b.SetPathAndCheck(filepath.Join(getLocalBinariesPath(), denoLocalName))

		checkOnStart := cfg.CheckUpdateOnStart

		go func() {
			b.Init()
			if checkOnStart {
				b.CheckUpdates()
			}
		}()
	} else {
		b.CancelDownload()
		b.SetPathAndCheck(cfg.DenoPath)
	}
}

func Get(name string) *DownloadableBinary {
	return downloadableMap[name]
}
