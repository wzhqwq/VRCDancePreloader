package interactive

import (
	"sync"
	"weak"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

type ManagedSettings[C any] interface {
	Current() C
	Update(func(*C) error, string) error
}

type setting[T any, C any] struct {
	field string
	mgr   ManagedSettings[C]

	get func(C) T
	set func(*C, T) error

	level *utils.Level[T]
}

func (s *setting[T, C]) Get() T {
	return s.get(s.mgr.Current())
}

func (s *setting[T, C]) Save(v T) error {
	err := s.mgr.Update(
		func(cfg *C) error {
			return s.set(cfg, v)
		},
		s.field,
	)
	if err != nil {
		return err
	}

	s.level.Store(v)
	return nil
}

// Watch hands the caller its own watcher of the setting's value level: wait on
// Wakes, then read Get (or the watcher's Current) for the value.
func (s *setting[T, C]) Watch() *utils.LevelWatcher[T] {
	return s.level.Subscribe()
}

// derivedLevel keeps a level of Out in sync with a setting of In: the value is
// re-derived every time the upstream setting is stored.
//
// The pump is started on the first Watch and then lives as long as the process.
// Settings are process-wide singletons (config.Manager getters over the weak
// cache below), so there is nothing to stop it for.
type derivedLevel[In any, Out any] struct {
	level    *utils.Level[Out]
	upstream StatefulSetting[In]
	derive   func() Out

	once sync.Once
}

func newDerivedLevel[In any, Out any](upstream StatefulSetting[In], derive func() Out) *derivedLevel[In, Out] {
	return &derivedLevel[In, Out]{
		level:    utils.NewLevel(derive()),
		upstream: upstream,
		derive:   derive,
	}
}

func (d *derivedLevel[In, Out]) watch() *utils.LevelWatcher[Out] {
	d.once.Do(func() {
		// This watcher (and the pump) belongs to the setting itself rather than to
		// one consumer, so it is never closed: a derived setting is built once per
		// config field and lives as long as the process.
		upstream := d.upstream.Watch()

		// The level was seeded when the setting was built, but the upstream can
		// have moved since without a pump running. Re-derive once here, before the
		// first subscriber starts reading Current().
		d.level.Store(d.derive())

		go func() {
			for {
				select {
				case <-upstream.Wakes():
					d.level.Store(d.derive())
				}
			}
		}()
	})

	return d.level.Subscribe()
}

type TypedSettingCache[T any, C any] struct {
	m ManagedSettings[C]

	cache map[string]weak.Pointer[setting[T, C]]
}

func (c TypedSettingCache[T, C]) NewSetting(
	field string,
	get func(C) T,
	set func(*C, T) error,
) StatefulSetting[T] {
	if s, ok := c.cache[field]; ok {
		if sv := s.Value(); sv != nil {
			return sv
		}
	}
	s := &setting[T, C]{
		mgr:   c.m,
		field: field,
		get:   get,
		set:   set,
		level: utils.NewLevel(get(c.m.Current())),
	}
	c.cache[field] = weak.Make(s)
	return s
}

type SettingCache[C any] struct {
	m ManagedSettings[C]

	stringCache     TypedSettingCache[string, C]
	intCache        TypedSettingCache[int, C]
	boolCache       TypedSettingCache[bool, C]
	stringListCache TypedSettingCache[[]string, C]
}

func NewSettingCache[C any](m ManagedSettings[C]) *SettingCache[C] {
	return &SettingCache[C]{
		m: m,

		stringCache:     TypedSettingCache[string, C]{m: m, cache: make(map[string]weak.Pointer[setting[string, C]])},
		intCache:        TypedSettingCache[int, C]{m: m, cache: make(map[string]weak.Pointer[setting[int, C]])},
		boolCache:       TypedSettingCache[bool, C]{m: m, cache: make(map[string]weak.Pointer[setting[bool, C]])},
		stringListCache: TypedSettingCache[[]string, C]{m: m, cache: make(map[string]weak.Pointer[setting[[]string, C]])},
	}
}

func (c *SettingCache[C]) NewStringSetting(
	field string,
	get func(C) string,
	set func(*C, string) error,
) StatefulSetting[string] {
	return c.stringCache.NewSetting(field, get, set)
}

func (c *SettingCache[C]) NewIntSetting(
	field string,
	get func(C) int,
	set func(*C, int) error,
) StatefulSetting[int] {
	return c.intCache.NewSetting(field, get, set)
}

func (c *SettingCache[C]) NewBoolSetting(
	field string,
	get func(C) bool,
	set func(*C, bool) error,
) StatefulSetting[bool] {
	return c.boolCache.NewSetting(field, get, set)
}

func (c *SettingCache[C]) NewStringListSetting(
	field string,
	get func(C) []string,
	set func(*C, []string) error,
) StatefulSetting[[]string] {
	return c.stringListCache.NewSetting(field, get, set)
}

type readonlyDerivedSetting[In any, Out any] struct {
	get func(In) Out

	s StatefulSetting[In]

	derived *derivedLevel[In, Out]
}

// newReadonlyDerivedSetting is the only place that wires derived. A derived
// setting built without it panics on the first Watch (nil d.derived), so every
// constructor — including NewDerivedSetting — has to go through here.
func newReadonlyDerivedSetting[In any, Out any](s StatefulSetting[In], get func(In) Out) *readonlyDerivedSetting[In, Out] {
	d := &readonlyDerivedSetting[In, Out]{
		get: get,
		s:   s,
	}
	d.derived = newDerivedLevel(s, d.Get)

	return d
}

func NewReadonlyDerivedSetting[In any, Out any](s StatefulSetting[In], get func(In) Out) StatefulSetting[Out] {
	return newReadonlyDerivedSetting(s, get)
}

func (d *readonlyDerivedSetting[In, Out]) Get() Out {
	return d.get(d.s.Get())
}

func (d *readonlyDerivedSetting[In, Out]) Save(_ Out) error {
	panic("you are saving a readonly setting")
}

func (d *readonlyDerivedSetting[In, Out]) Watch() *utils.LevelWatcher[Out] {
	return d.derived.watch()
}

type derivedSetting[In any, Out any] struct {
	*readonlyDerivedSetting[In, Out]

	set func(Out) In
}

func (d *derivedSetting[In, Out]) Save(t Out) error {
	return d.s.Save(d.set(t))
}

func NewDerivedSetting[In any, Out any](s StatefulSetting[In], get func(In) Out, set func(Out) In) StatefulSetting[Out] {
	return &derivedSetting[In, Out]{
		readonlyDerivedSetting: newReadonlyDerivedSetting(s, get),
		set:                    set,
	}
}
