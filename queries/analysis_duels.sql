-- name: CreateAnalysisDuel :one
-- Returns the inserted rowid so the persistence layer can map detector
-- LocalIDs to database primary keys when wiring analysis_mistakes.duel_id.
INSERT INTO analysis_duels (
    demo_id, round_number, round_id,
    attacker_steam, victim_steam,
    start_tick, end_tick,
    outcome, end_reason,
    hit_confirmed, hurt_count, shot_count,
    extras_json
)
VALUES (
    @demo_id, @round_number, @round_id,
    @attacker_steam, @victim_steam,
    @start_tick, @end_tick,
    @outcome, @end_reason,
    @hit_confirmed, @hurt_count, @shot_count,
    @extras_json
)
RETURNING id;

-- name: UpdateAnalysisDuelMutual :exec
-- Backfills mutual_duel_id once peer rows have ids. Runs in the same tx
-- as CreateAnalysisDuel so the link is atomic with insertion.
UPDATE analysis_duels SET mutual_duel_id = @mutual_duel_id WHERE id = @id;

-- name: DeleteAnalysisDuelsByDemoID :exec
DELETE FROM analysis_duels
WHERE demo_id = @demo_id;

-- name: ListAnalysisDuelsByDemoID :many
SELECT id, demo_id, round_number, round_id,
       attacker_steam, victim_steam,
       start_tick, end_tick,
       outcome, end_reason,
       hit_confirmed, hurt_count, shot_count,
       mutual_duel_id, extras_json, created_at
FROM analysis_duels
WHERE demo_id = @demo_id
ORDER BY round_number ASC, start_tick ASC, id ASC;

-- name: ListAnalysisDuelsByDemoIDAndSteamID :many
-- Returns duels where the supplied steamID is either the attacker or the
-- victim, sorted chronologically. Powers the per-player duel lane on the
-- viewer's round timeline.
SELECT id, demo_id, round_number, round_id,
       attacker_steam, victim_steam,
       start_tick, end_tick,
       outcome, end_reason,
       hit_confirmed, hurt_count, shot_count,
       mutual_duel_id, extras_json, created_at
FROM analysis_duels
WHERE demo_id = @demo_id
  AND (attacker_steam = @steam_id OR victim_steam = @steam_id)
ORDER BY round_number ASC, start_tick ASC, id ASC;

-- name: GetAnalysisDuelByID :one
SELECT id, demo_id, round_number, round_id,
       attacker_steam, victim_steam,
       start_tick, end_tick,
       outcome, end_reason,
       hit_confirmed, hurt_count, shot_count,
       mutual_duel_id, extras_json, created_at
FROM analysis_duels
WHERE id = @id;

-- name: ListAnalysisMistakesByDuelID :many
-- Used by GetDuelContext to materialise the tooltip / popover content
-- (the mistakes inside a duel) with one query.
SELECT id, demo_id, steam_id, round_number, round_id, tick, kind, category, severity, extras_json, duel_id, created_at
FROM analysis_mistakes
WHERE duel_id = @duel_id
ORDER BY tick ASC, id ASC;
