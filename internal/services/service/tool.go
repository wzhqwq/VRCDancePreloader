package service

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/stability"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type ConfigurableTool struct {
	Service

	name   string
	logger utils.LoggerImpl

	status interactive.RunnerStatus

	initializer func() error
	destroyer   func() error
}

func ConstructStatelessTool() ConfigurableTool {
	return ConfigurableTool{
		name: "stateless",
	}
}

func ConstructConfigurableTool(name string, initializer func() error, destroyer func() error, logger utils.LoggerImpl) ConfigurableTool {
	return ConfigurableTool{
		name:   name,
		logger: logger,

		initializer: initializer,
		destroyer:   destroyer,
	}
}

func (t *ConfigurableTool) Start() {
	if t.initializer == nil {
		t.status = interactive.RunnerStatus{
			Running: true,
		}
		return
	}
	err := t.initializer()
	if err != nil {
		t.status = interactive.RunnerStatus{
			Error: err,
		}
	} else {
		t.status = interactive.RunnerStatus{
			Running: true,
		}
	}
}

func (t *ConfigurableTool) Shutdown() error {
	if t.destroyer != nil {
		cancel := stability.PanicIfTimeout(t.name + "_ShuttingDown")
		defer cancel()

		t.logger.InfoLn("Shutting down...")
		defer t.logger.InfoLn("Shut down")

		return t.destroyer()
	}
	return nil
}

func (t *ConfigurableTool) Status() interactive.RunnerStatus {
	return t.status
}
