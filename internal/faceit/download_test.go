package faceit

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ok2ju/oversite/internal/demo"
	"github.com/ok2ju/oversite/internal/store"
	"github.com/ok2ju/oversite/internal/testutil"
)

// fakeDemContent is minimal content that passes the demo magic-bytes check.
// CS2 demos start with "PBDEMS2\x00".
var fakeDemContent = append([]byte("PBDEMS2\x00"), make([]byte, 1024)...)

func newTestDownloadService(t *testing.T, serverURL string) (*DownloadService, *store.Queries, store.User) {
	t.Helper()
	q, db := testutil.NewTestQueries(t)
	ctx := context.Background()
	user, err := q.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "faceit-dl-test", Nickname: "dl-tester",
		AvatarUrl: "", FaceitElo: 2000, FaceitLevel: 10, Country: "US",
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	downloadDir := t.TempDir()
	importSvc := demo.NewImportService(q, db)

	svc := NewDownloadService(
		&http.Client{},
		importSvc,
		q,
		downloadDir,
	)

	return svc, q, user
}

func seedFaceitMatchWithURL(t *testing.T, q *store.Queries, userID int64, demoURL string) store.FaceitMatch {
	t.Helper()
	m, err := q.CreateFaceitMatch(context.Background(), store.CreateFaceitMatchParams{
		UserID: userID, FaceitMatchID: "match-dl-1", MapName: "de_dust2",
		ScoreTeam: 16, ScoreOpponent: 10, Result: "win",
		DemoUrl: demoURL, PlayedAt: "2026-04-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateFaceitMatch: %v", err)
	}
	return m
}

func TestDownloadAndImport_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1032")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeDemContent)
	}))
	defer server.Close()

	svc, q, user := newTestDownloadService(t, server.URL)
	match := seedFaceitMatchWithURL(t, q, user.ID, server.URL+"/demo.dem")

	imported, err := svc.DownloadAndImport(context.Background(), match.ID, user.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imported == nil {
		t.Fatal("expected imported demo, got nil")
	}
	if imported.Status != "imported" {
		t.Errorf("Status = %q, want imported", imported.Status)
	}

	// Verify the Faceit match is now linked to the demo.
	updated, err := q.GetFaceitMatchByID(context.Background(), match.ID)
	if err != nil {
		t.Fatalf("GetFaceitMatchByID: %v", err)
	}
	if !updated.DemoID.Valid {
		t.Error("expected DemoID to be set after import")
	}
	if updated.DemoID.Int64 != imported.ID {
		t.Errorf("DemoID = %d, want %d", updated.DemoID.Int64, imported.ID)
	}
}

func TestDownloadAndImport_GzipDecompression(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		gz := gzip.NewWriter(w)
		_, _ = gz.Write(fakeDemContent)
		_ = gz.Close()
	}))
	defer server.Close()

	svc, q, user := newTestDownloadService(t, server.URL)
	match := seedFaceitMatchWithURL(t, q, user.ID, server.URL+"/demo.dem.gz")

	imported, err := svc.DownloadAndImport(context.Background(), match.ID, user.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imported == nil {
		t.Fatal("expected imported demo")
	}

	// Verify the final file exists and has .dem extension.
	finalPath := filepath.Join(svc.downloadDir, "faceit_match-dl-1.dem")
	if _, err := os.Stat(finalPath); err != nil {
		t.Errorf("expected final .dem file at %s: %v", finalPath, err)
	}
}

func TestDownloadAndImport_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	svc, q, user := newTestDownloadService(t, server.URL)
	m, err := q.CreateFaceitMatch(context.Background(), store.CreateFaceitMatchParams{
		UserID: user.ID, FaceitMatchID: "match-srv-err", MapName: "de_dust2",
		ScoreTeam: 16, ScoreOpponent: 10, Result: "win",
		DemoUrl: server.URL + "/demo.dem", PlayedAt: "2026-04-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateFaceitMatch: %v", err)
	}

	_, err = svc.DownloadAndImport(context.Background(), m.ID, user.ID, nil)
	if err == nil {
		t.Fatal("expected error for server error response")
	}
}

func TestDownloadAndImport_NoDemoURL(t *testing.T) {
	svc, q, user := newTestDownloadService(t, "")

	m, err := q.CreateFaceitMatch(context.Background(), store.CreateFaceitMatchParams{
		UserID: user.ID, FaceitMatchID: "match-no-url", MapName: "de_dust2",
		ScoreTeam: 16, ScoreOpponent: 10, Result: "win",
		DemoUrl: "", PlayedAt: "2026-04-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateFaceitMatch: %v", err)
	}

	_, err = svc.DownloadAndImport(context.Background(), m.ID, user.ID, nil)
	if err == nil {
		t.Fatal("expected error for match with no demo URL")
	}
}

func TestDownloadAndImport_Progress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1032")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeDemContent)
	}))
	defer server.Close()

	svc, q, user := newTestDownloadService(t, server.URL)
	seedFaceitMatchWithURL(t, q, user.ID, server.URL+"/demo.dem")

	match, _ := q.GetFaceitMatchesByUserID(context.Background(), store.GetFaceitMatchesByUserIDParams{
		UserID: user.ID, LimitVal: 1,
	})

	var progressCalls int
	_, err := svc.DownloadAndImport(context.Background(), match[0].ID, user.ID,
		func(_, _ int64) {
			progressCalls++
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if progressCalls == 0 {
		t.Error("expected progress callback to be called")
	}
}

func TestDownloadAndImport_MatchNotFound(t *testing.T) {
	svc, _, _ := newTestDownloadService(t, "")

	_, err := svc.DownloadAndImport(context.Background(), 9999, 1, nil)
	if err == nil {
		t.Fatal("expected error for non-existent match")
	}
}

func TestDownloadAndImport_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	svc, q, user := newTestDownloadService(t, server.URL)

	m, err := q.CreateFaceitMatch(context.Background(), store.CreateFaceitMatchParams{
		UserID: user.ID, FaceitMatchID: "match-http-err", MapName: "de_dust2",
		ScoreTeam: 16, ScoreOpponent: 10, Result: "win",
		DemoUrl: server.URL + "/demo.dem", PlayedAt: "2026-04-10T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("CreateFaceitMatch: %v", err)
	}

	_, err = svc.DownloadAndImport(context.Background(), m.ID, user.ID, nil)
	if err == nil {
		t.Fatal("expected error for HTTP error response")
	}
}

func TestDecompressGzip(t *testing.T) {
	dir := t.TempDir()

	// Create a gzipped file.
	gzPath := filepath.Join(dir, "test.dem.gz")
	gzFile, err := os.Create(gzPath)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	gz := gzip.NewWriter(gzFile)
	original := []byte("test demo content for decompression")
	_, _ = gz.Write(original)
	_ = gz.Close()
	_ = gzFile.Close()

	outPath, err := decompressGzip(gzPath, dir)
	if err != nil {
		t.Fatalf("decompressGzip: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != string(original) {
		t.Errorf("content = %q, want %q", string(content), string(original))
	}
}
