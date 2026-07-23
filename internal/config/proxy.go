package config

import (
	"github.com/wzhqwq/VRCDancePreloader/internal/utils/interactive"
)

func (m *Manager) ProxyOfSite(field string) interactive.StatefulSetting[string] {
	return m.cache.NewStringSetting(
		"proxy."+field,
		func(c Config) string {
			return c.Proxy.Value(field)
		},
		func(c *Config, value string) (err error) {
			c.Proxy, err = c.Proxy.Set(field, value)
			return
		},
	)
}
