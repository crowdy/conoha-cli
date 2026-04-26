package proxy

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// fakeExec is a programmable Executor whose responses depend on which class
// of command is being run (inspect / healthz / logs). The polling structure
// of WaitForHealthy is "inspect, then maybe healthz" per tick, so the tests
// shape a slice of `tick`s and consume one per iteration of the loop.
type tick struct {
	status   string // "" => simulate inspect failure (container missing)
	healthOK bool   // healthz returns ok on this tick
}

type fakeExec struct {
	ticks      []tick
	idx        int  // advances on inspect calls (each call is "one tick")
	logs       string
	sleepCount int
}

func (f *fakeExec) sleep(time.Duration) { f.sleepCount++ }

// currentTick is what inspect/healthz observe right now. After the slice is
// exhausted we keep returning the LAST entry — represents "the world settled
// into this state and stays there", which makes timeout tests deterministic
// regardless of how many polling iterations the no-op sleeper allows in.
func (f *fakeExec) currentTick() tick {
	if len(f.ticks) == 0 {
		return tick{} // empty — inspect will return missing
	}
	if f.idx < len(f.ticks) {
		return f.ticks[f.idx]
	}
	return f.ticks[len(f.ticks)-1]
}

func (f *fakeExec) Run(cmd string, _ io.Reader, stdout io.Writer) error {
	switch {
	case strings.Contains(cmd, "docker inspect"):
		t := f.currentTick()
		f.idx++
		if t.status == "" {
			return errors.New("inspect: container missing")
		}
		_, _ = fmt.Fprintln(stdout, t.status)
		return nil
	case strings.Contains(cmd, "/admin.sock"):
		// healthz reads the latest tick's healthOK without advancing the
		// cursor. WaitForHealthy only reaches this branch after the
		// stability counter trips, i.e. immediately after a successful
		// "running" inspect — so the previous tick is the relevant one.
		idx := f.idx - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(f.ticks) {
			idx = len(f.ticks) - 1
		}
		if idx < 0 || !f.ticks[idx].healthOK {
			return errors.New("healthz failed")
		}
		_, _ = io.WriteString(stdout, `{"status":"ok"}`)
		return nil
	case strings.Contains(cmd, "docker logs"):
		_, _ = io.WriteString(stdout, f.logs)
		return nil
	default:
		return fmt.Errorf("fakeExec: unexpected cmd %q", cmd)
	}
}

func TestWaitForHealthy_HappyPath(t *testing.T) {
	// 3 stable running samples + healthz ok on the third.
	f := &fakeExec{ticks: []tick{
		{status: "running", healthOK: true},
		{status: "running", healthOK: true},
		{status: "running", healthOK: true},
	}}
	err := WaitForHealthy(f, "conoha-proxy", "/var/lib/conoha-proxy", 30*time.Second, f.sleep,
		HealthcheckOptions{PollInterval: time.Millisecond, StableSamples: 3})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	// Two sleeps between three samples (sleep happens after deadline check
	// at end of iteration). The third inspect triggers healthz and returns
	// before the next sleep.
	if f.sleepCount != 2 {
		t.Errorf("sleeps: got %d, want 2", f.sleepCount)
	}
}

func TestWaitForHealthy_RecoversAfterRestart(t *testing.T) {
	// Restart loop for two ticks, then stabilizes. Counter must reset on
	// restart — if it didn't, healthcheck would false-positive on the brief
	// "running" windows during a crashloop.
	f := &fakeExec{ticks: []tick{
		{status: "running"},
		{status: "restarting"}, // resets counter
		{status: "running"},
		{status: "running"},
		{status: "running", healthOK: true},
	}}
	err := WaitForHealthy(f, "conoha-proxy", "/var/lib/conoha-proxy", 30*time.Second, f.sleep,
		HealthcheckOptions{PollInterval: time.Millisecond, StableSamples: 3})
	if err != nil {
		t.Fatalf("want nil after recovery, got %v", err)
	}
}

func TestWaitForHealthy_TimesOutOnRestartLoop(t *testing.T) {
	// Container bounces forever. With a tiny timeout we get a timeout error
	// containing the last status and the docker logs tail.
	ticks := make([]tick, 100)
	for i := range ticks {
		if i%2 == 0 {
			ticks[i] = tick{status: "running"}
		} else {
			ticks[i] = tick{status: "restarting"}
		}
	}
	f := &fakeExec{
		ticks: ticks,
		logs:  "Error: listen tcp :80: bind: permission denied\n",
	}
	err := WaitForHealthy(f, "conoha-proxy", "/var/lib/conoha-proxy", 5*time.Millisecond, f.sleep,
		HealthcheckOptions{PollInterval: time.Millisecond, StableSamples: 3})
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	for _, want := range []string{
		"did not become healthy",
		"5ms",
		"Last", // "Last 20 lines of `docker logs ..."
		"bind: permission denied",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q: %v", want, err)
		}
	}
}

func TestWaitForHealthy_TimesOutOnHealthzFailure(t *testing.T) {
	// Container is running but admin socket isn't responding (proxy's HTTP
	// listener died). Stability gate trips, healthz fails, counter holds —
	// but healthz keeps failing → timeout.
	ticks := make([]tick, 100)
	for i := range ticks {
		// Explicitly populate — zero-value tick has status="" which the fake
		// maps to "missing" via the inspect-error branch, defeating this
		// test's premise.
		ticks[i].status = "running"
		ticks[i].healthOK = false
	}
	f := &fakeExec{ticks: ticks, logs: "panic in HTTP handler\n"}
	err := WaitForHealthy(f, "conoha-proxy", "/var/lib/conoha-proxy", 3*time.Millisecond, f.sleep,
		HealthcheckOptions{PollInterval: time.Millisecond, StableSamples: 3})
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	if !strings.Contains(err.Error(), `"running"`) {
		t.Errorf("want last status=running in error; got: %v", err)
	}
}

func TestWaitForHealthy_MissingContainer(t *testing.T) {
	// `docker inspect` errors out (container never started). Status maps to
	// "missing", counter never trips, eventual timeout.
	ticks := make([]tick, 100)
	// All inspect calls fail (status "" => returns error).
	f := &fakeExec{ticks: ticks, logs: ""}
	err := WaitForHealthy(f, "conoha-proxy", "/var/lib/conoha-proxy", 2*time.Millisecond, f.sleep,
		HealthcheckOptions{PollInterval: time.Millisecond, StableSamples: 3})
	if err == nil {
		t.Fatal("want timeout error, got nil")
	}
	if !strings.Contains(err.Error(), `"missing"`) {
		t.Errorf("want last status=missing; got: %v", err)
	}
}

func TestShellQuote_HandlesEmbeddedQuote(t *testing.T) {
	got := shellQuote("a'b")
	want := `'a'\''b'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
