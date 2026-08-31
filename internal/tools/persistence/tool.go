package persistence

import "github.com/wzhqwq/VRCDancePreloader/internal/services/service"

type Tool struct {
	service.ConfigurableTool
}

func New() *Tool {
	return &Tool{
		ConfigurableTool: service.ConstructConfigurableTool("db", InitMainDB, CloseMainDB, logger),
	}
}
