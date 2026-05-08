package demo

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/ok2ju/oversite/internal/sysinfo"
)

// heapWatchdog polls runtime.ReadMemStats on a fixed interval *outside* the
// demoinfocs FrameDone handler. The in-handler heartbeat in parser.go can't
// fire while the underlying library is stuck in pre-frame work (string
// tables, entity baselines, DataTable decoding) — exactly the path where a
// pathological demo blows past the kill-switch on Windows. This watchdog
// runs as an independent goroutine so the kill-switch is enforced even while
// the parser dispatcher hasn't yielded control yet.
//
// On trip the watchdog writes a heap pprof profile to disk, calls onTrip
// (which sets state.limitExceeded and Cancels the parser), and stops itself
// — it does not loop. The FrameDone heartbeat keeps running as belt-and-
// braces for healthy parses where the pre-frame work has already completed.
type heapWatchdog struct {
	limit       uint64
	interval    time.Duration
	profileDir  string
	demoID      int64
	onTrip      func(error)
	warnOnce    sync.Once // soft-warning at 50% of limit fires at most once per parse
	dumpOnce    sync.Once // pprof dump and onTrip fire at most once per parse
	stopCh      chan struct{}
	doneCh      chan struct{}
	tripped     bool
	maxHeapSeen uint64
}

// newHeapWatchdog allocates a watchdog. Call Run in a goroutine to start it
// and Stop (e.g. via defer) to wind it down at the end of Parse. limit==0
// disables the watchdog (Run returns immediately).
//
// profileDir may be empty — in that case the watchdog still trips, just
// without writing a heap dump (it logs the missing-dir as a WARN so the
// user can find out why). demoID is included in the dump filename so a
// triage workflow can correlate the file with errors.txt.
func newHeapWatchdog(limit uint64, interval time.Duration, profileDir string, demoID int64, onTrip func(error)) *heapWatchdog {
	return &heapWatchdog{
		limit:      limit,
		interval:   interval,
		profileDir: profileDir,
		demoID:     demoID,
		onTrip:     onTrip,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
}

// Run polls memory stats and trips when the heap exceeds the limit. Designed
// to be invoked as `go wd.Run()`. Stop closes the stop channel and blocks
// until Run has returned (so callers can use it safely with defer).
func (w *heapWatchdog) Run() {
	defer close(w.doneCh)
	if w.limit == 0 || w.interval <= 0 {
		return
	}
	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-t.C:
			if w.poll() {
				return
			}
		}
	}
}

// Stop signals Run to exit and waits for it to do so. Safe to call multiple
// times; only the first close affects state.
func (w *heapWatchdog) Stop() {
	select {
	case <-w.stopCh:
		// already stopped
	default:
		close(w.stopCh)
	}
	<-w.doneCh
}

// poll reads memstats once and returns true if the watchdog has tripped (the
// caller should exit). Internal so tests can drive a single iteration.
func (w *heapWatchdog) poll() bool {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	procMem, _ := sysinfo.ProcessMemory()

	if mem.HeapAlloc > w.maxHeapSeen {
		w.maxHeapSeen = mem.HeapAlloc
	}

	if mem.HeapAlloc > w.limit/2 {
		w.warnOnce.Do(func() {
			slog.Warn("parser: heap usage is high",
				"heap_alloc_mb", mem.HeapAlloc>>20,
				"limit_mb", w.limit>>20,
				"working_set_mb", procMem.WorkingSetSize>>20,
			)
		})
	}

	heapTrip := mem.HeapAlloc > w.limit
	// On Windows, working-set may stay high even after the Go heap shrinks
	// because the runtime is slow to scavenge unused pages back to the OS.
	// 1.5x the heap limit catches the case where the heap looks "fine" but
	// the OS-visible memory has run away.
	wsTrip := procMem.WorkingSetSize > 0 && procMem.WorkingSetSize > w.limit+w.limit/2

	if !heapTrip && !wsTrip {
		return false
	}

	w.tripped = true
	w.dumpOnce.Do(func() {
		dumpPath := w.writeHeapProfile()
		reason := "heap_alloc"
		if wsTrip && !heapTrip {
			reason = "working_set"
		}
		err := fmt.Errorf("%w (limit %d MiB, observed heap %d MiB, working set %d MiB)",
			ErrHeapLimitExceeded, w.limit>>20, mem.HeapAlloc>>20, procMem.WorkingSetSize>>20)
		slog.Warn("parser: heap watchdog tripped; cancelling parse",
			"reason", reason,
			"heap_alloc_mb", mem.HeapAlloc>>20,
			"heap_sys_mb", mem.HeapSys>>20,
			"heap_idle_mb", mem.HeapIdle>>20,
			"heap_inuse_mb", mem.HeapInuse>>20,
			"stack_inuse_mb", mem.StackInuse>>20,
			"sys_mb", mem.Sys>>20,
			"next_gc_mb", mem.NextGC>>20,
			"num_gc", mem.NumGC,
			"pause_total_ns", mem.PauseTotalNs,
			"working_set_mb", procMem.WorkingSetSize>>20,
			"private_usage_mb", procMem.PrivateUsage>>20,
			"limit_mb", w.limit>>20,
			"profile_path", dumpPath,
		)
		if w.onTrip != nil {
			w.onTrip(err)
		}
	})
	return true
}

// writeHeapProfile dumps a pprof heap profile to {profileDir}/heap-{demoID}-{ts}.pprof.
// Returns the path on success or "" on failure (logs the failure as WARN so
// it surfaces in errors.txt, but does not block the trip).
func (w *heapWatchdog) writeHeapProfile() string {
	if w.profileDir == "" {
		slog.Warn("parser: heap watchdog tripped but no profile dir configured; skipping pprof dump")
		return ""
	}
	name := fmt.Sprintf("heap-%d-%d.pprof", w.demoID, time.Now().UnixNano())
	path := filepath.Join(w.profileDir, name)
	f, err := os.Create(path)
	if err != nil {
		slog.Warn("parser: failed to create heap profile file", "path", path, "err", err)
		return ""
	}
	defer f.Close() //nolint:errcheck
	if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
		slog.Warn("parser: failed to write heap profile", "path", path, "err", err)
		return ""
	}
	return path
}
