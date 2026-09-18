package service

import (
	"net"
	"net/http"
	"testing"
	"time"
)

// selfTestControl is both the Control and the ServerLike of the test service, so
// that it can start a server through ServeAndTest and shut it down again.
//
// It answers every request with 500, which is the cheapest way to make the self
// check fail.
type selfTestControl struct {
	svc  *BaseService[struct{}]
	port int
	srv  *http.Server
}

func (c *selfTestControl) Enabled() bool { return true }

func (c *selfTestControl) ServiceStart() error {
	c.srv = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	}

	// This is the shape mixed_server and live use.
	return c.svc.ServeAndTest(c.port, c)
}

func (c *selfTestControl) Serve(l net.Listener) error { return c.srv.Serve(l) }

func (c *selfTestControl) ServiceStop() error {
	if c.srv == nil {
		return nil
	}
	return c.srv.Close()
}

// freePort reserves a port and releases it again, so ServeAndTest can bind it.
func freePort(t *testing.T) int {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a port: %v", err)
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port
}

// P1-1 — a failed self check must not make the service invisible.
//
// ServeAndTest reports the failure by setting lastError and returning nil: the
// service is up and stays up. That only works if Start keeps the error (it used
// to clear it right after ServiceStart) and if Status reports it next to
// Running (it used to drop it whenever the service was running).
//
// It guards the other direction too: an earlier attempt at this fix called
// s.Stop() from inside ServeAndTest, which deadlocked, because Start calls
// ServiceStart while holding s.mu and Stop locks that same mutex. The timeout
// below is what catches that.
func TestFailedSelfCheckStaysVisibleWhileRunning(t *testing.T) {
	svc := &BaseService[struct{}]{}
	ctrl := &selfTestControl{svc: svc, port: freePort(t)}
	svc.SetControl("Self Test", ctrl)

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.Start()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return: the self check teardown deadlocked")
	}

	// Safe from here on: Start has returned, so it is no longer holding s.mu.
	t.Cleanup(func() { svc.Stop() })

	status := svc.Status()

	if !status.Running {
		t.Fatal("a failed self check must not interrupt the start flow")
	}

	// Split the two ways the error used to disappear, and report both instead of
	// stopping at the first: a failure then says exactly which one came back.
	if svc.lastError == nil {
		t.Error("Start discarded the error recorded during ServiceStart")
	}
	if status.Error == nil {
		t.Error("Status does not report the error while the service is running")
	}
}
