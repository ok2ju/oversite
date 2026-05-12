//go:build calibration

// Calibration harness for the Phase 5 severity-tuning loop
// (.claude/plans/timeline-contact-moments/phase-5/01-calibration-tooling.md).
//
// Operator runs:
//
//	go test -tags=calibration -timeout=30m -v \
//	  ./internal/demo/contacts/detectors/ \
//	  -run TestSeverityDistribution \
//	  -corpus=$(pwd)/testdata/corpus
//
// The harness ingests every .dem under -corpus through the production
// pipeline (parser.Parse → IngestRounds → IngestPlayerVisibility →
// contacts.Run → detectors.Run) and emits
// docs/knowledge/contact-mistake-severity-calibration.md with per-kind
// severity histograms. Hand-edited RATIONALE blocks survive across
// runs.
package detectors_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/demo/contacts/detectors"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

var corpusFlag = flag.String(
	"corpus", "",
	"absolute path to a directory containing .dem files for calibration",
)

const perDemoTimeout = 2 * time.Minute

type corpusEntry struct {
	Filename  string
	SHA12     string
	Bytes     int64
	Rounds    int
	Contacts  int
	Mistakes  int
	SkipError string
}

type bucket struct {
	low, medium, high int
}

func (b bucket) total() int { return b.low + b.medium + b.high }

func severityLabel(sev int) string {
	switch sev {
	case 1:
		return "low"
	case 2:
		return "medium"
	case 3:
		return "high"
	default:
		return ""
	}
}

func TestSeverityDistribution(t *testing.T) {
	if *corpusFlag == "" {
		t.Skip("set -corpus=/abs/path to a folder of .dem files")
	}

	entries, err := os.ReadDir(*corpusFlag)
	if err != nil {
		t.Fatalf("read corpus dir %q: %v", *corpusFlag, err)
	}

	var demos []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
			continue
		}
		if !strings.EqualFold(filepath.Ext(name), ".dem") {
			continue
		}
		demos = append(demos, filepath.Join(*corpusFlag, name))
	}
	sort.Strings(demos)

	if len(demos) == 0 {
		t.Fatalf("no .dem files found under %q", *corpusFlag)
	}

	corpus := make([]corpusEntry, 0, len(demos))
	counts := make(map[string]*bucket)
	totalMistakes := 0

	for i, path := range demos {
		t.Logf("[%d/%d] %s", i+1, len(demos), filepath.Base(path))
		entry, mistakes, skipMsg := runOneDemo(t, path)
		entry.Filename = filepath.Base(path)
		if skipMsg != "" {
			entry.SkipError = skipMsg
			corpus = append(corpus, entry)
			continue
		}
		corpus = append(corpus, entry)

		for _, m := range mistakes {
			label := severityLabel(m.Mistake.Severity)
			if label == "" {
				t.Errorf("empty severity for kind=%s (severity=%d)", m.Mistake.Kind, m.Mistake.Severity)
				continue
			}
			b := counts[m.Mistake.Kind]
			if b == nil {
				b = &bucket{}
				counts[m.Mistake.Kind] = b
			}
			switch label {
			case "low":
				b.low++
			case "medium":
				b.medium++
			case "high":
				b.high++
			}
			totalMistakes++
		}
	}

	reportPath, err := reportPath()
	if err != nil {
		t.Fatalf("locate report path: %v", err)
	}

	report := renderMarkdown(corpus, counts, totalMistakes, *corpusFlag)
	report = preserveRationaleBlocks(reportPath, report)
	if err := os.WriteFile(reportPath, []byte(report), 0o644); err != nil {
		t.Fatalf("write report %q: %v", reportPath, err)
	}
	t.Logf("wrote %s", reportPath)

	// Invariants — a regression in the detector pipeline (a kind silently
	// producing zero rows) fails the harness.
	if totalMistakes < 100 {
		t.Errorf("total mistakes = %d, want >= 100 (sanity)", totalMistakes)
	}
	for _, e := range detectors.V1() {
		if e.Kind == "bad_crosshair_height" {
			// Pre-pitch demos legitimately produce zero of these. Carve-out
			// documented in Phase 3 03-detectors-pre.md §6.
			continue
		}
		if counts[e.Kind] == nil || counts[e.Kind].total() == 0 {
			t.Errorf("kind %q has zero rows in corpus — likely Phase 3 regression", e.Kind)
		}
	}
}

func runOneDemo(t *testing.T, path string) (corpusEntry, []detectors.BoundContactMistake, string) {
	t.Helper()
	entry := corpusEntry{}

	info, err := os.Stat(path)
	if err != nil {
		return entry, nil, fmt.Sprintf("stat: %v", err)
	}
	entry.Bytes = info.Size()

	sum, err := sha256File(path)
	if err != nil {
		return entry, nil, fmt.Sprintf("sha256: %v", err)
	}
	entry.SHA12 = sum

	ctx, cancel := context.WithTimeout(context.Background(), perDemoTimeout)
	defer cancel()

	f, err := os.Open(path)
	if err != nil {
		return entry, nil, fmt.Sprintf("open: %v", err)
	}
	defer func() { _ = f.Close() }()

	parser := demo.NewDemoParser()
	result, err := parser.Parse(ctx, f)
	if err != nil && result == nil {
		return entry, nil, fmt.Sprintf("parse: %v", err)
	}
	if result == nil {
		return entry, nil, "parse: nil result"
	}
	entry.Rounds = len(result.Rounds)

	db := testutil.NewTestDB(t)

	// Insert a demos row so the FK on rounds.demo_id is satisfied.
	q := store.New(db)
	row, err := q.CreateDemo(ctx, store.CreateDemoParams{
		MapName:   result.Header.MapName,
		FilePath:  path,
		FileSize:  entry.Bytes,
		Status:    "ready",
		MatchDate: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return entry, nil, fmt.Sprintf("insert demo: %v", err)
	}
	demoID := row.ID

	roundMap, err := demo.IngestRounds(ctx, db, demoID, result)
	if err != nil {
		return entry, nil, fmt.Sprintf("ingest rounds: %v", err)
	}
	if _, err := demo.IngestGameEvents(ctx, db, demoID, result.Events, roundMap); err != nil {
		return entry, nil, fmt.Sprintf("ingest events: %v", err)
	}
	if _, err := demo.IngestPlayerVisibility(ctx, db, demoID, result.Visibility, roundMap); err != nil {
		return entry, nil, fmt.Sprintf("ingest visibility: %v", err)
	}

	contactsList, err := contacts.Run(result, roundMap, contacts.RunOpts{})
	if err != nil {
		return entry, nil, fmt.Sprintf("contacts.Run: %v", err)
	}
	if err := contacts.Persist(ctx, db, demoID, contactsList); err != nil {
		return entry, nil, fmt.Sprintf("contacts.Persist: %v", err)
	}
	entry.Contacts = len(contactsList)

	// Force=true so the runner doesn't short-circuit on
	// MaxDetectorVersionForDemo (the in-memory DB has no prior rows but
	// the gate still queries; Force keeps it deterministic).
	bound, _, _, err := detectors.Run(ctx, db, demoID, result, contactsList, detectors.RunOpts{Force: true})
	if err != nil {
		return entry, nil, fmt.Sprintf("detectors.Run: %v", err)
	}
	entry.Mistakes = len(bound)
	return entry, bound, ""
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil))[:12], nil
}

func renderMarkdown(corpus []corpusEntry, counts map[string]*bucket, totalMistakes int, corpusDir string) string {
	var b strings.Builder

	totalContacts := 0
	parsed := 0
	for _, e := range corpus {
		if e.SkipError == "" {
			parsed++
			totalContacts += e.Contacts
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	fmt.Fprintf(&b, "# Contact-mistake severity calibration (v1)\n\n")
	fmt.Fprintf(&b, "> Generated: %s by `go test -tags=calibration ...`\n", now)
	fmt.Fprintf(&b, "> Corpus: %d demos under `%s`\n", parsed, corpusDir)
	fmt.Fprintf(&b, "> Total contacts: %s · Total mistakes: %s\n\n",
		humanInt(totalContacts), humanInt(totalMistakes))
	b.WriteString("<!-- DO NOT HAND-EDIT THE TABLES BELOW. -->\n")
	b.WriteString("<!-- They are overwritten on every harness run. -->\n")
	b.WriteString("<!-- Hand-write rationale inside the per-kind RATIONALE blocks. -->\n\n")

	if parsed < 20 {
		fmt.Fprintf(&b, "> ⚠️ **Warning:** Only %d demos in corpus (need 20+ for confidence).\n", parsed)
		b.WriteString("> Distributions are directional, not definitive.\n\n")
	}

	// --- Corpus table ---
	b.WriteString("## Corpus\n\n")
	b.WriteString("| # | filename | sha256 (first 12) | bytes | rounds | contacts | mistakes |\n")
	b.WriteString("|---|----------|-------------------|-------|--------|----------|----------|\n")
	for i, e := range corpus {
		if e.SkipError != "" {
			fmt.Fprintf(&b, "| %d | %s | %s | %s | — | — | — (skipped: %s) |\n",
				i+1, e.Filename, e.SHA12, humanBytes(e.Bytes), e.SkipError)
			continue
		}
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %d | %d | %d |\n",
			i+1, e.Filename, e.SHA12, humanBytes(e.Bytes), e.Rounds, e.Contacts, e.Mistakes)
	}
	b.WriteString("\n")

	// --- Per-kind distribution tables ---
	b.WriteString("## Per-kind distribution\n\n")
	kinds := make([]detectors.Entry, 0, len(detectors.V1()))
	kinds = append(kinds, detectors.V1()...)
	sort.SliceStable(kinds, func(i, j int) bool {
		if kinds[i].Phase != kinds[j].Phase {
			return phaseRank(kinds[i].Phase) < phaseRank(kinds[j].Phase)
		}
		return kinds[i].Kind < kinds[j].Kind
	})

	for _, e := range kinds {
		fmt.Fprintf(&b, "### `%s`\n\n", e.Kind)
		c := counts[e.Kind]
		if c == nil {
			c = &bucket{}
		}
		total := c.total()
		b.WriteString("| severity | count | pct |\n")
		b.WriteString("|----------|-------|-----|\n")
		fmt.Fprintf(&b, "| low      | %5d | %s |\n", c.low, pct(c.low, total))
		fmt.Fprintf(&b, "| medium   | %5d | %s |\n", c.medium, pct(c.medium, total))
		fmt.Fprintf(&b, "| high     | %5d | %s |\n", c.high, pct(c.high, total))
		b.WriteString("\n")
		fmt.Fprintf(&b, "**Default today:** %s (per Phase 3 catalog).\n\n", severityLabel(e.Severity))
		fmt.Fprintf(&b, "<!-- BEGIN RATIONALE: %s -->\n", e.Kind)
		b.WriteString("_To be filled by the operator after reviewing the distribution._\n")
		fmt.Fprintf(&b, "<!-- END RATIONALE: %s -->\n\n", e.Kind)
	}

	// --- Summary ---
	var totLow, totMed, totHigh int
	for _, c := range counts {
		totLow += c.low
		totMed += c.medium
		totHigh += c.high
	}
	grand := totLow + totMed + totHigh
	b.WriteString("## Summary\n\n")
	b.WriteString("| target distribution | actual |\n")
	b.WriteString("|---------------------|--------|\n")
	fmt.Fprintf(&b, "| low ≈ 70%%           | %s |\n", pct(totLow, grand))
	fmt.Fprintf(&b, "| medium ≈ 25%%        | %s |\n", pct(totMed, grand))
	fmt.Fprintf(&b, "| high ≈ 5%%           | %s |\n", pct(totHigh, grand))
	b.WriteString("\n")

	b.WriteString("## Operator notes\n\n")
	b.WriteString("_Filled in by hand. See plan file `.claude/plans/timeline-contact-moments/phase-5/02-severity-tuning.md` §10.2._\n\n")
	b.WriteString("## Open issues\n\n")
	b.WriteString("_Filled in by hand. See `02-severity-tuning.md` §10.3._\n")

	return b.String()
}

func phaseRank(p detectors.Phase) int {
	switch p {
	case detectors.PhasePre:
		return 0
	case detectors.PhaseDuring:
		return 1
	case detectors.PhasePost:
		return 2
	default:
		return 3
	}
}

func pct(n, total int) string {
	if total == 0 {
		return "  0%"
	}
	return fmt.Sprintf("%3d%%", (n*100+total/2)/total)
}

func humanInt(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var parts []string
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

func humanBytes(n int64) string {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)
	switch {
	case n >= GB:
		return fmt.Sprintf("%.1fG", float64(n)/GB)
	case n >= MB:
		return fmt.Sprintf("%.1fM", float64(n)/MB)
	case n >= KB:
		return fmt.Sprintf("%.1fK", float64(n)/KB)
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// reportPath walks up from this test file to the repo root and joins
// docs/knowledge/contact-mistake-severity-calibration.md.
func reportPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "docs", "knowledge", "contact-mistake-severity-calibration.md"), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("could not find go.mod from %s", file)
}

// preserveRationaleBlocks merges the operator-written content of every
// <!-- BEGIN RATIONALE: kind --> ... <!-- END RATIONALE: kind --> block
// from the on-disk report into the freshly rendered one.
func preserveRationaleBlocks(path, fresh string) string {
	existing, err := os.ReadFile(path)
	if err != nil {
		return fresh
	}
	re := regexp.MustCompile(`(?ms)<!-- BEGIN RATIONALE: (\S+) -->(.*?)<!-- END RATIONALE: \1 -->`)
	old := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(string(existing), -1) {
		old[m[1]] = m[2]
	}
	out := re.ReplaceAllStringFunc(fresh, func(match string) string {
		sub := re.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		kind := sub[1]
		body, ok := old[kind]
		if !ok {
			return match
		}
		return fmt.Sprintf("<!-- BEGIN RATIONALE: %s -->%s<!-- END RATIONALE: %s -->", kind, body, kind)
	})
	return out
}
