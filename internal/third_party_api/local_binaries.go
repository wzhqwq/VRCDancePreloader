package third_party_api

import (
	"io/fs"
	"path"
	"path/filepath"

	"github.com/wzhqwq/VRCDancePreloader/internal/gui/custom_fyne"
)

func getLocalBinary(name string) (fs.File, bool) {
	file, err := binaries.Open(filepath.Join(getLocalBinariesPath(), name))
	return file, err == nil
}

func getLocalBinariesPath() string {
	if custom_fyne.AppDataRoot == "" {
		// not in fyne mode, use relative path
		return "./binaries"
	}
	return path.Join(custom_fyne.AppDataRoot, "binaries")
}

func InitBinaries() {
	loadEmbeddedVersions()
	loadVersions()
}
