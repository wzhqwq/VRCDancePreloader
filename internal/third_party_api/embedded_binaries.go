package third_party_api

import (
	"embed"
	"errors"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

//go:embed binaries
var binaries embed.FS

var embeddedVersions BinaryVersions

func loadEmbeddedVersions() {
	versionsFile, err := binaries.Open(filepath.Join(getLocalBinariesPath(), "versions.yaml"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return
		}
		logger.FatalLnf("Open embedded versions.yaml error: %s", err)
	}
	defer versionsFile.Close()

	decoder := yaml.NewDecoder(versionsFile)
	err = decoder.Decode(&localVersions)
	if err != nil {
		logger.FatalLnf("Failed to parse embedded versions.yaml: %s", err)
	}
}

func getBinary(name string) (fs.File, bool) {
	file, err := binaries.Open(filepath.Join("binaries", name))
	return file, err == nil
}

func getEmbeddedYtDlp() (fs.File, bool) {
	return getBinary("yt-dlp.exe")
}
