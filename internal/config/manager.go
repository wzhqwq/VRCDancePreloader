package config

import (
	"strings"
	"sync"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

type ApplyFunc func(config Config, fields []string) error

type Manager struct {
	mu    sync.RWMutex
	cfg   Config
	apply ApplyFunc

	cache *interactive.SettingCache[Config]
}

func NewManager(cfg Config, apply ApplyFunc) *Manager {
	m := &Manager{
		cfg:   cfg,
		apply: apply,
	}
	m.cache = interactive.NewSettingCache[Config](m)

	return m
}

func (m *Manager) Current() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Fill(cfg Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.apply != nil {
		if err := m.apply(cfg, nil); err != nil {
			return err
		}
	}

	if err := SaveFileAtomic(cfg); err != nil {
		return err
	}

	m.cfg = cfg
	return nil
}

func (m *Manager) update(mutator func(*Config) error, fields []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	next := m.cfg

	if err := mutator(&next); err != nil {
		return err
	}

	if m.apply != nil {
		if err := m.apply(next, fields); err != nil {
			return err
		}
	}

	if err := SaveFileAtomic(next); err != nil {
		return err
	}

	m.cfg = next
	return nil
}

func (m *Manager) Update(mutator func(*Config) error, field string) error {
	fields := strings.SplitN(field, ".", 2)

	if err := m.update(mutator, fields); err != nil {
		return err
	}
	return nil
}
