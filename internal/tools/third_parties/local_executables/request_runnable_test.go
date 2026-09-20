package local_executables

import (
	"errors"
	"os"
	"testing"
	"time"
)

// lockIsHeld reports whether d.mutex is currently held, by asking a writer for
// it. A held read lock blocks a writer, which is exactly the symptom of the
// leaked lock this file is about.
//
// The writer goroutine is left blocked on purpose when the lock is held; the
// caller releases it later and the goroutine finishes on its own.
func lockIsHeld(d *DownloadableBinary) bool {
	acquired := make(chan struct{})

	go func() {
		d.mutex.Lock()
		close(acquired)
		d.mutex.Unlock()
	}()

	select {
	case <-acquired:
		return false
	case <-time.After(time.Second):
		return true
	}
}

// P2-1 — the failure path must release the read lock it took.
//
// RequestRunnable used to keep the lock when there was nothing to run and rely on
// the caller's unconditional ReleaseRunnable to drop it. That paired up in the
// one call site that existed, but any caller written the obvious way
//
//	if err := d.RequestRunnable(); err != nil {
//		return err
//	}
//	defer d.ReleaseRunnable()
//
// left the mutex read locked forever, and every later Lock() blocked in silence.
func TestRequestRunnableDoesNotLeakTheReadLockWhenThereIsNothingToRun(t *testing.T) {
	d := &DownloadableBinary{} // no path, so nothing is runnable

	if err := d.RequestRunnable(); !errors.Is(err, ErrExecutableNotFound) {
		t.Fatalf("RequestRunnable = %v, want ErrExecutableNotFound", err)
	}

	if lockIsHeld(d) {
		t.Fatal("RequestRunnable failed but kept the read lock: every later Lock() would block")
	}
}

// The successful path still holds the lock until the caller releases it, and
// exactly one release is enough. Together with the test above this pins the pair.
func TestRequestRunnableHoldsTheLockUntilReleased(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("locate the test binary: %v", err)
	}

	d := &DownloadableBinary{path: self}

	if err := d.RequestRunnable(); err != nil {
		t.Fatalf("RequestRunnable = %v, want nil for an existing executable", err)
	}

	if !lockIsHeld(d) {
		t.Fatal("a successful request has to hold the read lock until ReleaseRunnable")
	}

	d.ReleaseRunnable()

	if lockIsHeld(d) {
		t.Fatal("ReleaseRunnable did not release the lock")
	}
}
