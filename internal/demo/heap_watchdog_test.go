package demo

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHeapWatchdog_RunReturnsImmediatelyWhenLimitIsZero(t *testing.T) {
	wd := newHeapWatchdog(0, 100*time.Millisecond, "", 0, func(error) {
		t.Fatal("onTrip must not fire when limit is zero")
	})

	done := make(chan struct{})
	go func() {
		wd.Run()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return when limit is zero")
	}
}

func TestHeapWatchdog_PollDoesNotTripBelowLimit(t *testing.T) {
	wd := newHeapWatchdog(1<<60, time.Millisecond, "", 0, func(error) {
		t.Fatal("onTrip must not fire below limit")
	})

	if got := wd.poll(); got {
		t.Errorf("poll() = true under huge limit, want false")
	}
	if wd.tripped {
		t.Errorf("tripped = true under huge limit, want false")
	}
}

func TestHeapWatchdog_PollTripsAboveLimit(t *testing.T) {
	dir := t.TempDir()

	var (
		gotErr atomic.Pointer[error]
		calls  atomic.Int32
	)
	wd := newHeapWatchdog(1, time.Millisecond, dir, 42, func(err error) {
		calls.Add(1)
		gotErr.Store(&err)
	})

	if got := wd.poll(); !got {
		t.Fatalf("poll() = false above tiny limit, want true")
	}
	if !wd.tripped {
		t.Errorf("tripped = false after poll trip, want true")
	}
	if calls.Load() != 1 {
		t.Errorf("onTrip call count = %d, want 1", calls.Load())
	}
	errp := gotErr.Load()
	if errp == nil || *errp == nil {
		t.Fatalf("onTrip received nil error")
	}
	if !errors.Is(*errp, ErrHeapLimitExceeded) {
		t.Errorf("error = %v, want errors.Is(...) ErrHeapLimitExceeded", *errp)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q): %v", dir, err)
	}
	var pprofFiles []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".pprof" {
			pprofFiles = append(pprofFiles, e.Name())
		}
	}
	if len(pprofFiles) != 1 {
		t.Fatalf("pprof file count = %d, want 1 (entries: %v)", len(pprofFiles), entries)
	}
	if !strings.HasPrefix(pprofFiles[0], "heap-42-") {
		t.Errorf("pprof filename = %q, want prefix heap-42-", pprofFiles[0])
	}
	info, err := os.Stat(filepath.Join(dir, pprofFiles[0]))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("pprof file is empty")
	}

	// Re-polling must not call onTrip again — sync.Once guards both the
	// callback and the dump so a tight ticker doesn't spam.
	if got := wd.poll(); !got {
		t.Errorf("second poll() = false, want true (still over limit)")
	}
	if calls.Load() != 1 {
		t.Errorf("onTrip call count after second poll = %d, want 1 (sync.Once)", calls.Load())
	}
}

func TestHeapWatchdog_PollWithoutProfileDirSkipsDump(t *testing.T) {
	var calls atomic.Int32
	wd := newHeapWatchdog(1, time.Millisecond, "", 0, func(error) {
		calls.Add(1)
	})

	if got := wd.poll(); !got {
		t.Fatalf("poll() = false above tiny limit, want true")
	}
	if calls.Load() != 1 {
		t.Errorf("onTrip call count = %d, want 1 (callback should fire even without profile dir)", calls.Load())
	}
}

func TestHeapWatchdog_StopIsIdempotent(t *testing.T) {
	wd := newHeapWatchdog(1<<60, 10*time.Millisecond, "", 0, nil)
	go wd.Run()

	wd.Stop()
	wd.Stop() // must not panic on a closed channel
}

func TestHeapWatchdog_RunStopsOnStop(t *testing.T) {
	wd := newHeapWatchdog(1<<60, 10*time.Millisecond, "", 0, nil)

	done := make(chan struct{})
	go func() {
		wd.Run()
		close(done)
	}()

	// Let the ticker fire at least once before stopping.
	time.Sleep(20 * time.Millisecond)
	wd.Stop()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not exit after Stop")
	}
}
