package interactive

import (
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

	em *utils.EventManager[T]
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

	s.em.NotifySubscribers(v)
	return nil
}

func (s *setting[T, C]) Subscribe() *utils.EventSubscriber[T] {
	return s.em.SubscribeEvent()
}

func (s *setting[T, C]) SubscribeWhether(whether func(T) bool) *utils.EventSubscriber[bool] {
	return utils.PipeEvent(s.em, func(in T) (bool, bool) {
		return whether(in), true
	})
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
		em:    utils.NewEventManager[T](),
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

type derivedSetting[In any, Out any] struct {
	get func(In) Out
	set func(Out) In

	s StatefulSetting[In]
}

func (d *derivedSetting[In, Out]) SubscribeWhether(whether func(Out) bool) *utils.EventSubscriber[bool] {
	return d.s.SubscribeWhether(func(in In) bool {
		return whether(d.get(in))
	})
}

func (d *derivedSetting[In, Out]) Get() Out {
	return d.get(d.s.Get())
}

func (d *derivedSetting[In, Out]) Save(t Out) error {
	return d.s.Save(d.set(t))
}

func (d *derivedSetting[In, Out]) Subscribe() *utils.EventSubscriber[Out] {
	return utils.PipeSubEvent(d.s.Subscribe(), func(in In) (Out, bool) {
		return d.get(in), true
	})
}

func NewDerivedSetting[In any, Out any](s StatefulSetting[In], get func(In) Out, set func(Out) In) StatefulSetting[Out] {
	return &derivedSetting[In, Out]{
		get: get,
		set: set,
		s:   s,
	}
}
