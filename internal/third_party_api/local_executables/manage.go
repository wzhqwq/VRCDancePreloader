package local_executables

import (
	"path/filepath"
)

var downloadableMap = map[string]*DownloadableBinary{
	"ytdlp": NewDownloadableBinary("ytdlp"),
	"deno":  NewDownloadableBinary("deno"),
}

func InitYtDlp(autoCheck bool, path string) {
	b := downloadableMap["ytdlp"]
	if path == "<vrcdp>" {
		b.SetPathAndCheck(filepath.Join(getLocalBinariesPath(), ytDlpLocalName))
		go func() {
			b.Init()
			if autoCheck {
				b.CheckUpdates()
			}
		}()
	} else {
		b.CancelDownload()
		b.SetPathAndCheck(path)
	}
}
func InitDeno(autoCheck bool, path string) {
	b := downloadableMap["deno"]
	if path == "<vrcdp>" {
		b.SetPathAndCheck(filepath.Join(getLocalBinariesPath(), denoLocalName))
		go func() {
			b.Init()
			if autoCheck {
				b.CheckUpdates()
			}
		}()
	} else {
		b.CancelDownload()
		b.SetPathAndCheck(path)
	}
}

func Get(name string) *DownloadableBinary {
	return downloadableMap[name]
}

func Stop() {
	downloadableMap["ytdlp"].Stop()
	downloadableMap["deno"].Stop()
}
