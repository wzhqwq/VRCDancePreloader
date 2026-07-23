package watcher

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

func initialize() error {
	roaming, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("getting user config directory: %w", err)
	}
	base := filepath.Join(roaming, "..", "LocalLow", "VRChat", "VRChat")

	// check if the log directory exists first
	_, err = os.Stat(base)
	if err != nil {
		return err
	}

	logBase = base
	// start watching the log directory
	path, err := sniffActiveLog()
	if err == nil {
		if keepTrackUntilClose(path) != nil {
			return err
		}
	}

	wg.Go(func() {
		err = watch()
		if err != nil {
			logger.ErrorLn("Error watching directory:", err)
			panic(err)
		}
	})

	return nil
}

func destroy() error {
	// stop watching the log directory
	if dirWatcher != nil {
		dirWatcher.Close()
	}
	if watchingFile != nil {
		watchingFile.Close()
	}
	wg.Wait()
	return nil
}

type Tool struct {
	service.ConfigurableTool
}

func New() *Tool {
	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool(initialize, destroy),
	}
}
