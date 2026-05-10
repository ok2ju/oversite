package store

// Manual implementations for analysis_mistakes queries that sqlc's SQLite
// parser cannot accept (e.g. JOIN-on-subquery with named parameters used in
// both the inner and outer scopes). Mirrors the heatmaps_custom.go pattern.

import "context"

const listMistakeKindCountsForPlayerLookback = `
SELECT am.kind, count(*) AS total
FROM analysis_mistakes am
JOIN (
    SELECT pma.demo_id
    FROM player_match_analysis pma
    JOIN demos d ON d.id = pma.demo_id
    WHERE pma.steam_id = ?
    ORDER BY d.match_date DESC, pma.demo_id DESC
    LIMIT ?
) rd ON rd.demo_id = am.demo_id
WHERE am.steam_id = ?
GROUP BY am.kind
ORDER BY total DESC, am.kind ASC
`

// ListMistakeKindCountsForPlayerLookbackParams is the parameter struct for
// the manually implemented coaching errors-strip query.
type ListMistakeKindCountsForPlayerLookbackParams struct {
	SteamID  string
	LimitVal int64
}

// ListMistakeKindCountsForPlayerLookbackRow mirrors the (kind, total) shape
// the coaching errors strip consumes — one row per mistake kind with a count
// > 0 across the player's last N analyzed demos.
type ListMistakeKindCountsForPlayerLookbackRow struct {
	Kind  string
	Total int64
}

// ListMistakeKindCountsForPlayerLookback counts mistakes per kind for a
// player across their last N analyzed demos (N = LimitVal, ordered by
// match_date DESC). Powers the coaching errors strip; one round-trip per
// coaching report.
func (q *Queries) ListMistakeKindCountsForPlayerLookback(ctx context.Context, arg ListMistakeKindCountsForPlayerLookbackParams) ([]ListMistakeKindCountsForPlayerLookbackRow, error) {
	rows, err := q.db.QueryContext(ctx, listMistakeKindCountsForPlayerLookback,
		arg.SteamID, arg.LimitVal, arg.SteamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var items []ListMistakeKindCountsForPlayerLookbackRow
	for rows.Next() {
		var i ListMistakeKindCountsForPlayerLookbackRow
		if err := rows.Scan(&i.Kind, &i.Total); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
