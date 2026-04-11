package handler

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/auth"
	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/worker"
)

// FaceitStore is the subset of store.Queries needed by FaceitHandler.
type FaceitStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (store.User, error)
	GetEloHistory(ctx context.Context, arg store.GetEloHistoryParams) ([]store.GetEloHistoryRow, error)
	CountFaceitMatchesByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
	GetCurrentStreak(ctx context.Context, userID uuid.UUID) ([]string, error)
	CountFaceitMatchesFiltered(ctx context.Context, arg store.CountFaceitMatchesFilteredParams) (int64, error)
	GetFaceitMatchesFiltered(ctx context.Context, arg store.GetFaceitMatchesFilteredParams) ([]store.FaceitMatch, error)
}

// FaceitDemoImporter is the subset of faceit.DemoImporter needed by FaceitHandler.
type FaceitDemoImporter interface {
	ImportByMatchID(ctx context.Context, userID, matchID uuid.UUID) (*faceit.ImportResult, error)
}

// FaceitHandler handles Faceit-related HTTP endpoints.
type FaceitHandler struct {
	queue    JobEnqueuer
	store    FaceitStore
	importer FaceitDemoImporter
}

// NewFaceitHandler creates a new FaceitHandler.
func NewFaceitHandler(queue JobEnqueuer, store FaceitStore, importer FaceitDemoImporter) *FaceitHandler {
	return &FaceitHandler{queue: queue, store: store, importer: importer}
}

// HandleSync enqueues a Faceit match history sync job for the authenticated user.
// Returns 202 Accepted on success.
func (h *FaceitHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	faceitID, ok := auth.FaceitIDFromContext(r.Context())
	if !ok {
		slog.Error("faceit_id not found in context", "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "faceit_id not available"})
		return
	}

	if _, err := h.queue.Enqueue(r.Context(), worker.FaceitSyncStream, map[string]interface{}{
		"user_id":   userIDStr,
		"faceit_id": faceitID,
	}); err != nil {
		slog.Error("enqueueing faceit sync job", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to enqueue sync job"})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "sync_queued"})
}

// mapResult maps raw DB result values ("W", "L") to frontend-expected strings.
func mapResult(raw string) string {
	switch raw {
	case "W":
		return "win"
	case "L":
		return "loss"
	default:
		return raw
	}
}

// streakResult computes the current streak from a list of recent match results.
func streakResult(results []string) (string, int) {
	if len(results) == 0 {
		return "none", 0
	}
	first := results[0]
	count := 0
	for _, r := range results {
		if r != first {
			break
		}
		count++
	}
	return mapResult(first), count
}

type profileResponse struct {
	Data profileData `json:"data"`
}

type profileData struct {
	Nickname      string        `json:"nickname"`
	AvatarURL     *string       `json:"avatar_url"`
	Elo           *int32        `json:"elo"`
	Level         *int16        `json:"level"`
	Country       *string       `json:"country"`
	MatchesPlayed int64         `json:"matches_played"`
	CurrentStreak currentStreak `json:"current_streak"`
}

type currentStreak struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// HandleGetProfile returns the authenticated user's Faceit profile data.
func (h *FaceitHandler) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
		return
	}

	user, err := h.store.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}
		slog.Error("getting user", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	matchCount, err := h.store.CountFaceitMatchesByUserID(r.Context(), userID)
	if err != nil {
		slog.Error("counting matches", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	recentResults, err := h.store.GetCurrentStreak(r.Context(), userID)
	if err != nil {
		slog.Error("getting streak", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	streakType, streakCount := streakResult(recentResults)

	pd := profileData{
		Nickname:      user.Nickname,
		MatchesPlayed: matchCount,
		CurrentStreak: currentStreak{Type: streakType, Count: streakCount},
	}
	if user.AvatarUrl.Valid {
		pd.AvatarURL = &user.AvatarUrl.String
	}
	if user.FaceitElo.Valid {
		pd.Elo = &user.FaceitElo.Int32
	}
	if user.FaceitLevel.Valid {
		pd.Level = &user.FaceitLevel.Int16
	}
	if user.Country.Valid {
		pd.Country = &user.Country.String
	}

	writeJSON(w, http.StatusOK, profileResponse{Data: pd})
}

type eloHistoryResponse struct {
	Data []eloHistoryPoint `json:"data"`
}

type eloHistoryPoint struct {
	Elo      *int32 `json:"elo"`
	MapName  string `json:"map_name"`
	PlayedAt string `json:"played_at"`
}

// HandleGetEloHistory returns the authenticated user's ELO history.
func (h *FaceitHandler) HandleGetEloHistory(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user ID"})
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		d, err := strconv.Atoi(daysStr)
		if err != nil || (d != 30 && d != 90 && d != 180 && d != 0) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "days must be 0, 30, 90, or 180"})
			return
		}
		days = d
	}

	var since time.Time
	if days > 0 {
		since = time.Now().AddDate(0, 0, -days)
	}

	rows, err := h.store.GetEloHistory(r.Context(), store.GetEloHistoryParams{
		UserID:   userID,
		PlayedAt: since,
	})
	if err != nil {
		slog.Error("getting elo history", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	points := make([]eloHistoryPoint, 0, len(rows))
	for _, row := range rows {
		p := eloHistoryPoint{
			MapName:  row.MapName,
			PlayedAt: row.PlayedAt.Format(time.RFC3339),
		}
		if row.EloAfter.Valid {
			p.Elo = &row.EloAfter.Int32
		}
		points = append(points, p)
	}

	writeJSON(w, http.StatusOK, eloHistoryResponse{Data: points})
}

// HandleGetMatches returns a paginated, filtered list of Faceit matches for the authenticated user.
func (h *FaceitHandler) HandleGetMatches(w http.ResponseWriter, r *http.Request) {
	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
		return
	}

	page, perPage := parsePagination(r)

	var mapFilter, resultFilter sql.NullString
	if v := r.URL.Query().Get("map_name"); v != "" {
		mapFilter = sql.NullString{String: v, Valid: true}
	}
	if v := r.URL.Query().Get("result"); v != "" {
		if v != "W" && v != "L" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid result filter: must be W or L"})
			return
		}
		resultFilter = sql.NullString{String: v, Valid: true}
	}

	countArg := store.CountFaceitMatchesFilteredParams{
		UserID:  userID,
		MapName: mapFilter,
		Result:  resultFilter,
	}
	total, err := h.store.CountFaceitMatchesFiltered(r.Context(), countArg)
	if err != nil {
		slog.Error("counting faceit matches", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to count matches"})
		return
	}

	listArg := store.GetFaceitMatchesFilteredParams{
		UserID:  userID,
		Limit:   int32(perPage),
		Offset:  int32((page - 1) * perPage),
		MapName: mapFilter,
		Result:  resultFilter,
	}
	matches, err := h.store.GetFaceitMatchesFiltered(r.Context(), listArg)
	if err != nil {
		slog.Error("listing faceit matches", "error", err, "request_id", chimw.GetReqID(r.Context()))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list matches"})
		return
	}

	items := make([]faceitMatchResponse, 0, len(matches))
	for _, m := range matches {
		items = append(items, faceitMatchToJSON(m))
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": items,
		"meta": map[string]interface{}{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

type faceitMatchResponse struct {
	ID            string  `json:"id"`
	FaceitMatchID string  `json:"faceit_match_id"`
	MapName       string  `json:"map_name"`
	ScoreTeam     int16   `json:"score_team"`
	ScoreOpponent int16   `json:"score_opponent"`
	Result        string  `json:"result"`
	EloBefore     *int32  `json:"elo_before"`
	EloAfter      *int32  `json:"elo_after"`
	EloChange     *int32  `json:"elo_change"`
	Kills         *int16  `json:"kills"`
	Deaths        *int16  `json:"deaths"`
	Assists       *int16  `json:"assists"`
	DemoUrl       *string `json:"demo_url"`
	DemoID        *string `json:"demo_id"`
	HasDemo       bool    `json:"has_demo"`
	PlayedAt      string  `json:"played_at"`
}

func faceitMatchToJSON(m store.FaceitMatch) faceitMatchResponse {
	resp := faceitMatchResponse{
		ID:            m.ID.String(),
		FaceitMatchID: m.FaceitMatchID,
		MapName:       m.MapName,
		ScoreTeam:     m.ScoreTeam,
		ScoreOpponent: m.ScoreOpponent,
		Result:        m.Result,
		HasDemo:       m.DemoID.Valid,
		PlayedAt:      m.PlayedAt.Format("2006-01-02T15:04:05Z"),
	}
	if m.EloBefore.Valid {
		v := m.EloBefore.Int32
		resp.EloBefore = &v
	}
	if m.EloAfter.Valid {
		v := m.EloAfter.Int32
		resp.EloAfter = &v
	}
	if m.EloBefore.Valid && m.EloAfter.Valid {
		v := m.EloAfter.Int32 - m.EloBefore.Int32
		resp.EloChange = &v
	}
	if m.Kills.Valid {
		v := m.Kills.Int16
		resp.Kills = &v
	}
	if m.Deaths.Valid {
		v := m.Deaths.Int16
		resp.Deaths = &v
	}
	if m.Assists.Valid {
		v := m.Assists.Int16
		resp.Assists = &v
	}
	if m.DemoUrl.Valid {
		resp.DemoUrl = &m.DemoUrl.String
	}
	if m.DemoID.Valid {
		s := m.DemoID.UUID.String()
		resp.DemoID = &s
	}
	return resp
}

// HandleImportMatch imports a demo for a specific Faceit match.
// Returns 201 Created with demo_id and file_size on success.
func (h *FaceitHandler) HandleImportMatch(w http.ResponseWriter, r *http.Request) {
	if h.importer == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "demo import not available"})
		return
	}

	userIDStr, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid user id"})
		return
	}

	matchID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid match id"})
		return
	}

	result, err := h.importer.ImportByMatchID(r.Context(), userID, matchID)
	if err != nil {
		switch {
		case errors.Is(err, faceit.ErrMatchNotFound), errors.Is(err, faceit.ErrMatchForbidden):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "match not found"})
		case errors.Is(err, faceit.ErrNoDemoURL):
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "match has no demo URL"})
		case errors.Is(err, faceit.ErrDemoAlreadyLinked):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "demo already imported"})
		case errors.Is(err, faceit.ErrDownloadFailed):
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "demo download failed"})
		default:
			slog.Error("importing faceit demo", "error", err, "match_id", matchID, "request_id", chimw.GetReqID(r.Context()))
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "import failed"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"data": map[string]interface{}{
			"demo_id":   result.DemoID.String(),
			"file_size": result.FileSize,
		},
	})
}
