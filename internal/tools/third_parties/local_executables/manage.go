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
		go func() {
			b.Init()
			if cfg.CheckUpdateOnStart {
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
		go func() {
			b.Init()
			if cfg.CheckUpdateOnStart {
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
