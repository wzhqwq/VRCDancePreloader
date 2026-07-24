package host

import (
	"errors"

	"github.com/wzhqwq/VRCDancePreloader/internal/config"
	"github.com/wzhqwq/VRCDancePreloader/internal/services/service"
)

type CfgApplier interface {
	Validate(cfg config.Config, fields []string) error
	Update(cfg config.Config, fields []string) error
}

type FieldsFunc func(fields []string) error

type BaseCfgApplier struct {
	v FieldsFunc
	u FieldsFunc
}

func (a BaseCfgApplier) Validate(fields []string) error {
	return a.v(fields)
}

func (a BaseCfgApplier) Update(fields []string) error {
	return a.u(fields)
}

type NoFieldValidationCfg interface {
	Validate() error
}

type NoFieldCfgApplier[T NoFieldValidationCfg] struct {
	getter func(cfg config.Config) T
	u      func(cfg T) error
}

func (a NoFieldCfgApplier[T]) Validate(cfg config.Config, _ []string) error {
	return a.getter(cfg).Validate()
}

func (a NoFieldCfgApplier[T]) Update(cfg config.Config, _ []string) error {
	return a.u(a.getter(cfg))
}

type OneFieldValidationCfg interface {
	Validate(field string) error
}

type OneFieldCfgApplier[T OneFieldValidationCfg] struct {
	getter func(cfg config.Config) T
	u      func(cfg T, field string) error
}

func (a OneFieldCfgApplier[T]) Validate(cfg config.Config, fields []string) error {
	if len(fields) == 0 {
		return a.getter(cfg).Validate("")
	}
	return a.getter(cfg).Validate(fields[0])
}

func (a OneFieldCfgApplier[T]) Update(cfg config.Config, fields []string) error {
	if len(fields) == 0 {
		return a.u(a.getter(cfg), "")
	}
	return a.u(a.getter(cfg), fields[0])
}

func (m *Manager) RegisterConfiguredService(name string, s service.Service, updater CfgApplier, deps ...string) {
	m.RegisterService(name, s, deps...)
	m.cfgUpdaters[name] = updater
}

func (m *Manager) Init() {
	m.cfgMgr = config.NewManager(m.cfg, func(c config.Config, fields []string) error {
		if len(fields) == 0 {
			for _, updater := range m.cfgUpdaters {
				if err := updater.Validate(c, nil); err != nil {
					return err
				}
			}
			for _, updater := range m.cfgUpdaters {
				if err := updater.Update(c, nil); err != nil {
					return err
				}
			}
			return nil
		}

		updater, ok := m.cfgUpdaters[fields[0]]
		if !ok {
			return errors.New("no updater for " + fields[0])
		}

		if err := updater.Validate(c, fields[1:]); err != nil {
			return err
		}

		return updater.Update(c, fields[1:])
	})
}
