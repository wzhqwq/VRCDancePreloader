package catalog

import (
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

type Tool struct {
	service.ConfigurableTool
}

var wg sync.WaitGroup

func GetPyPyDanceCatalogManager() Manager[raw_song.PyPyDanceSong] {
	return &pypyCatalogManager
}

func GetWannaDanceCatalogManager() Manager[raw_song.WannaDanceSong] {
	return &wannaCatalogManager
}

func GetDuDuFitDanceCatalogManager() Manager[raw_song.DuDuFitDanceSong] {
	return &duduCatalogManager
}

func initialize() error {
	setupManagers()
	//updateAll()
	return nil
}

func destroy() error {
	pypyCatalogManager.shutdown()
	wannaCatalogManager.shutdown()
	duduCatalogManager.shutdown()
	wg.Wait()
	return nil
}

func New() *Tool {
	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool(initialize, destroy),
	}
}
