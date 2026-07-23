package catalog

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/services/cache_manager"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
	"github.com/wzhqwq/VRCDancePreloader/internal/song/raw_song"
)

type Tool struct {
	service.ConfigurableTool
}

func GetPyPyDanceCatalogManager() Manager[raw_song.PyPyDanceSong] {
	return &pypyCatalogManager
}

func GetWannaDanceCatalogManager() Manager[raw_song.WannaDanceSong] {
	return &wannaCatalogManager
}

func GetDuDuFitDanceCatalogManager() Manager[raw_song.DuDuFitDanceSong] {
	return &duduCatalogManager
}

func destroy() error {
	pypyCatalogManager.shutdown()
	wannaCatalogManager.shutdown()
	duduCatalogManager.shutdown()
	return nil
}

func New(cacheSvc *cache_manager.Service) *Tool {
	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool(func() error {
			setupManagers(cacheSvc)
			pypyCatalogManager.Update()
			wannaCatalogManager.Update()
			duduCatalogManager.Update()
			return nil
		}, destroy),
	}
}
