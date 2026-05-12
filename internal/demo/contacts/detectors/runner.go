// Package detectors runs the v1 mistake-detection rules over the
// persisted contact_moments rows for a demo and writes findings to
// contact_mistakes. The runner is invoked from app.go after Phase 2's
// contacts.Persist has committed.
//
// DetectorVersion is the compiled detector schema. Bumped when a
// detector's math changes, a new detector lands, or the catalog gains
// a v1 entry. The runner refuses to overwrite contact_mistakes rows
// at DetectorVersion >= current unless the caller forces a rebuild
// (RunOpts.Force == true).
package detectors

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/demo/analysis"
	"github.com/ok2ju/oversite/internal/demo/contacts"
	"github.com/ok2ju/oversite/internal/store"
)

// DetectorVersion is the compiled detector schema version stamped on
// every contact_mistakes row Phase 3 writes. Phase 2's default of 0
// means "never run"; a successful Phase 3 ingest moves the column.
const DetectorVersion = 1

// RunOpts controls runner behavior. Force=true bypasses the
// MaxDetectorVersionForDemo gate (used for debug / calibration).
type RunOpts struct {
	Force bool
}

// BoundContactMistake pairs a ContactMistake with the contact_id it
// should attach to. The runner produces a flat slice instead of a map
// so the persister can insert in deterministic order without
// re-sorting.
type BoundContactMistake struct {
	ContactID int64
	Mistake   ContactMistake
}

// Run executes all v1 detectors against every contact in the supplied
// list. The caller is expected to pass the post-persist Contact set
// (with ID populated by contacts.Persist) so the bound mistakes can
// be inserted with the correct FK.
//
// Returns:
//   - perContact: flat []BoundContactMistake keyed by contact_id +
//     kind + tick — the persister inserts these directly.
//   - aggregate: round-level analysis.Mistake rows for kinds with
//     WriteAggregate=true. The caller appends these to the slice
//     analysis.PersistWithRoundMap consumes.
//   - skipped: true when MaxDetectorVersionForDemo >= DetectorVersion
//     and opts.Force is false. Callers should NOT clear
//     contact_mistakes in this case.
//   - err: programmer / data-shape failures (nil result, sqlc query
//     failure, etc.). A demo with zero contacts returns
//     (nil, nil, false, nil).
func Run(
	ctx context.Context,
	db *sql.DB,
	demoID int64,
	result *demo.ParseResult,
	contactsList []contacts.ContactMoment,
	opts RunOpts,
) (perContact []BoundContactMistake, aggregate []analysis.Mistake, skipped bool, err error) {
	if result == nil {
		return nil, nil, false, fmt.Errorf("detectors: nil ParseResult")
	}

	if !opts.Force && db != nil {
		q := store.New(db)
		maxRaw, qerr := q.MaxDetectorVersionForDemo(ctx, demoID)
		if qerr != nil {
			return nil, nil, false, fmt.Errorf("max detector version: %w", qerr)
		}
		if coerceInt64(maxRaw) >= int64(DetectorVersion) {
			return nil, nil, true, nil
		}
	}

	if len(contactsList) == 0 {
		return nil, nil, false, nil
	}

	rd := NewRunData(result)
	bySubject := groupContactsBySubject(contactsList)
	subjects := make([]string, 0, len(bySubject))
	for s := range bySubject {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects)

	perContact = make([]BoundContactMistake, 0, 64)
	aggregate = make([]analysis.Mistake, 0, 32)

	for _, subject := range subjects {
		rows := bySubject[subject]
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].TFirst != rows[j].TFirst {
				return rows[i].TFirst < rows[j].TFirst
			}
			return rows[i].ID < rows[j].ID
		})
		var prevEnd int32 = -1
		for _, row := range rows {
			c := materializeContact(row)
			dctx := BuildCtx(c, prevEnd, rd)
			findings := runAllDetectors(c, dctx)
			for _, m := range findings.PerContact {
				perContact = append(perContact, BoundContactMistake{
					ContactID: row.ID,
					Mistake:   m,
				})
			}
			aggregate = append(aggregate, findings.Aggregate...)
			prevEnd = c.TLast
		}
	}

	return perContact, aggregate, false, nil
}

// detectorOutput is the per-contact findings the runner accumulates.
type detectorOutput struct {
	PerContact []ContactMistake
	Aggregate  []analysis.Mistake
}

// runAllDetectors invokes every V1() entry against c+ctx and bundles
// the findings, plus round-level aggregate rows for kinds with
// WriteAggregate=true.
func runAllDetectors(c *contacts.Contact, ctx *DetectorCtx) detectorOutput {
	var out detectorOutput
	for _, e := range V1() {
		ms := e.Func(c, ctx)
		if len(ms) == 0 {
			continue
		}
		out.PerContact = append(out.PerContact, ms...)
		if !e.WriteAggregate {
			continue
		}
		for _, m := range ms {
			tick := 0
			if m.Tick != nil {
				tick = int(*m.Tick)
			}
			out.Aggregate = append(out.Aggregate, analysis.Mistake{
				SteamID:     c.Subject,
				RoundNumber: c.RoundNumber,
				Tick:        tick,
				Kind:        m.Kind,
				Extras:      m.Extras,
			})
		}
	}
	return out
}

// materializeContact builds a *contacts.Contact from a persisted
// ContactMoment. Signals come straight through from the in-memory
// contact (Phase 2's Persist returns the list with ID populated and
// Signals still in place).
func materializeContact(row contacts.ContactMoment) *contacts.Contact {
	enemies := append([]string(nil), row.Enemies...)
	signals := append([]contacts.Signal(nil), row.Signals...)
	return &contacts.Contact{
		RoundNumber: row.RoundNumber,
		RoundID:     row.RoundID,
		Subject:     row.SubjectSteam,
		TFirst:      row.TFirst,
		TLast:       row.TLast,
		TPre:        row.TPre,
		TPost:       row.TPost,
		Enemies:     enemies,
		Extras:      row.Extras,
		Signals:     signals,
	}
}

// groupContactsBySubject collects contacts in a (subject → contacts)
// map so the runner can iterate per-subject and thread
// PreviousContactEnd through the chronological sequence.
func groupContactsBySubject(list []contacts.ContactMoment) map[string][]contacts.ContactMoment {
	out := make(map[string][]contacts.ContactMoment, 16)
	for _, c := range list {
		out[c.SubjectSteam] = append(out[c.SubjectSteam], c)
	}
	return out
}

// coerceInt64 normalizes the interface{} return of the sqlc-generated
// MaxDetectorVersionForDemo into an int64 (the COALESCE column comes
// out of SQLite as either int64 or nil). Returns 0 for nil.
func coerceInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case nil:
		return 0
	default:
		return 0
	}
}
