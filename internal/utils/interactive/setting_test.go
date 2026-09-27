package interactive

import (
	"strconv"
	"testing"
	"time"

	"github.com/wzhqwq/VRCDancePreloader/internal/utils"
)

// stubSetting is the smallest StatefulSetting: a level plus the write-through
// Save. It stands in for the config-backed settings the GUI builds.
type stubSetting[T any] struct {
	level *utils.Level[T]
}

func newStubSetting[T any](initial T) *stubSetting[T] {
	return &stubSetting[T]{level: utils.NewLevel(initial)}
}

func (s *stubSetting[T]) Get() T { return s.level.Current() }

func (s *stubSetting[T]) Save(value T) error {
	s.level.Store(value)
	return nil
}

func (s *stubSetting[T]) Watch() *utils.LevelWatcher[T] { return s.level.Subscribe() }

func waitWake[T any](t *testing.T, w *utils.LevelWatcher[T]) {
	t.Helper()
	select {
	case <-w.Wakes():
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a wake-up")
	}
}

// Both constructors must wire the derived level: a derived setting without it
// used to panic on the first Watch (nil d.derived), which is how NewDerivedSetting
// was found broken by hand rather than by a test.
func TestDerivedSettingConstructorsAreWatchable(t *testing.T) {
	upstream := newStubSetting(1)

	readonly := NewReadonlyDerivedSetting(upstream, func(v int) string {
		return strconv.Itoa(v)
	})
	w := readonly.Watch()
	defer w.Close()
	if got := w.Current(); got != "1" {
		t.Fatalf("readonly derived setting: got %q, want %q", got, "1")
	}

	writable := NewDerivedSetting(
		upstream,
		func(v int) string { return strconv.Itoa(v) },
		func(s string) int {
			v, err := strconv.Atoi(s)
			if err != nil {
				return -1
			}
			return v
		},
	)
	w2 := writable.Watch()
	defer w2.Close()
	if got := w2.Current(); got != "1" {
		t.Fatalf("derived setting: got %q, want %q", got, "1")
	}

	// Save goes the other way: through the setter into the upstream setting.
	if err := writable.Save("42"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := upstream.Get(); got != 42 {
		t.Fatalf("upstream after Save: got %d, want 42", got)
	}
	waitWake(t, w2)
	if got := w2.Current(); got != "42" {
		t.Fatalf("derived setting after Save: got %q, want %q", got, "42")
	}
}

// A wake-up is only "read the value again": the derived level must hold the new
// value by the time its subscribers are woken, so consumers can read
// watcher.Current() instead of recomputing the derivation.
func TestDerivedSettingAdvancesTheLevelBeforeWaking(t *testing.T) {
	upstream := newStubSetting(0)
	derived := NewReadonlyDerivedSetting(upstream, func(v int) bool { return v > 0 })

	w := derived.Watch()
	defer w.Close()
	if w.Current() {
		t.Fatal("initial value should be false for upstream 0")
	}

	upstream.Save(5)
	waitWake(t, w)
	if !w.Current() {
		t.Fatal("the stored value must be the new one once a watcher is woken")
	}
}

// The level is seeded when the setting is built, but the upstream can move before
// anyone watches: the first Watch has to re-derive instead of handing out the
// stale seed.
func TestDerivedSettingReDerivesOnFirstWatch(t *testing.T) {
	upstream := newStubSetting(1)
	derived := NewReadonlyDerivedSetting(upstream, func(v int) bool { return v > 0 })

	upstream.Save(-1)

	w := derived.Watch()
	defer w.Close()
	if w.Current() {
		t.Fatal("the first watcher must see the re-derived value, not the construction-time seed")
	}
}

// One watcher per consumer: a single upstream change must wake every consumer of
// the derived level, not just one of them.
func TestDerivedSettingWakesEveryWatcher(t *testing.T) {
	upstream := newStubSetting(0)
	derived := NewReadonlyDerivedSetting(upstream, func(v int) int { return v * 2 })

	first := derived.Watch()
	defer first.Close()
	second := derived.Watch()
	defer second.Close()

	if got := first.Current(); got != 0 {
		t.Fatalf("first watcher: got %d, want 0", got)
	}

	upstream.Save(3)

	waitWake(t, first)
	waitWake(t, second)

	if got := first.Current(); got != 6 {
		t.Fatalf("first watcher after store: got %d, want 6", got)
	}
	if got := second.Current(); got != 6 {
		t.Fatalf("second watcher after store: got %d, want 6", got)
	}
}
