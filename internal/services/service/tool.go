package service

import "github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"

type ConfigurableTool struct {
	Service

	status interactive.RunnerStatus

	initializer func() error
	destroyer   func() error
}

func ConstructConfigurableTool(initializer func() error, destroyer func() error) ConfigurableTool {
	return ConfigurableTool{
		initializer: initializer,
		destroyer:   destroyer,
	}
}

func NewConfigurableTool(initializer func() error, destroyer func() error) *ConfigurableTool {
	return &ConfigurableTool{
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
		return t.destroyer()
	}
	return nil
}

func (t *ConfigurableTool) Status() interactive.RunnerStatus {
	return t.status
}
