package watcher

import (
	"sort"
	"sync/atomic"
)

type versionedValue[T any] struct {
	version int32
	value   T
}

type LastValue[T any] struct {
	slot atomic.Pointer[versionedValue[T]]
}

func NewLastValue[T any](initial T) *LastValue[T] {
	lv := &LastValue[T]{}
	lv.Reset(initial)
	return lv
}

func (lv *LastValue[T]) Set(version int32, value T) bool {
	for {
		old := lv.slot.Load()
		if version <= old.version {
			return false
		}
		nw := &versionedValue[T]{version: version, value: value}
		if lv.slot.CompareAndSwap(old, old) {
			lv.slot.Store(nw)
			return true
		}
	}
}

func (lv *LastValue[T]) Get() T {
	cur := lv.slot.Load()
	return cur.value
}

func (lv *LastValue[T]) Reset(initial T) {
	lv.slot.Store(&versionedValue[T]{version: -1, value: initial})
}
func (lv *LastValue[T]) ResetVersion() {
	value := lv.slot.Load().value
	lv.slot.Store(&versionedValue[T]{version: -1, value: value})
}

type OrderedValues[T any] struct {
	values chan versionedValue[T]
}

func NewOrderedValues[T any]() *OrderedValues[T] {
	return &OrderedValues[T]{values: make(chan versionedValue[T], 10000)}
}

func (o *OrderedValues[T]) Add(version int32, value T) {
	o.values <- versionedValue[T]{version: version, value: value}
}

func (o *OrderedValues[T]) Flush() []versionedValue[T] {
	values := make([]versionedValue[T], 0, len(o.values))
	for v := range o.values {
		values = append(values, v)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].version < values[j].version
	})
	return values
}
