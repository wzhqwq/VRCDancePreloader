package local_executables

import (
	"context"
	"os/exec"
)

//type BinaryVersions struct {
//	YtDlp string `yaml:"yt-dlp"`
//}
//
//var versionsMutex = &sync.Mutex{}
//var localVersions BinaryVersions
//
//func loadVersions() {
//	_, err := os.Stat(filepath.Join(getLocalBinariesPath(), "versions.yaml"))
//	if errors.Is(err, os.ErrPermission) {
//		logger.FatalLn("versions.yaml permission denied")
//	}
//
//	if err == nil {
//		versionsFile, err := os.Open(filepath.Join(getLocalBinariesPath(), "versions.yaml"))
//		if err != nil {
//			logger.FatalLnf("Open versions.yaml error: %s", err)
//		}
//		defer versionsFile.Close()
//
//		decoder := yaml.NewDecoder(versionsFile)
//		err = decoder.Decode(&localVersions)
//		if err != nil {
//			logger.FatalLnf("Failed to parse versions.yaml: %s", err)
//		}
//	}
//}
//
//func saveVersions() {
//	versionsMutex.Lock()
//	defer versionsMutex.Unlock()
//
//	versionsFile, err := os.Create(filepath.Join(getLocalBinariesPath(), "versions.yaml"))
//	if err != nil {
//		logger.FatalLnf("Open or create versions.yaml error: %s", err)
//	}
//	defer versionsFile.Close()
//
//	encoder := yaml.NewEncoder(versionsFile)
//	err = encoder.Encode(&localVersions)
//	if err != nil {
//		logger.FatalLnf("Failed to save versions.yaml: %s", err)
//	}
//}

func execVersionCheck(name string, ctx context.Context) (string, bool) {
	executable, ok := getLocalBinary(name)
	if !ok {
		return "", false
	}

	cmd := exec.CommandContext(ctx, executable, "--version")
	output, err := cmd.Output()

	if err != nil {
		logger.ErrorLn("Failed to get version of", name, err)
		return "", false
	}

	return string(output), true
}
