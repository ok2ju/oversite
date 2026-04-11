//go:build integration

package faceit_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/storage"
	"github.com/ok2ju/oversite/backend/internal/store"
	"github.com/ok2ju/oversite/backend/internal/testutil"
	"github.com/ok2ju/oversite/backend/internal/worker"

	"github.com/redis/go-redis/v9"
)

const (
	minioAccessKey = "minioadmin"
	minioSecretKey = "minioadmin"
	minioBucket    = "import-test-bucket"
)

type integrationEnv struct {
	queries *store.Queries
	minio   *storage.MinIOClient
	queue   *worker.RedisQueue
}

func setupIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()
	ctx := context.Background()

	// Postgres
	pgContainer, dbURL, err := testutil.PostgresContainer(ctx)
	if err != nil {
		t.Fatalf("starting postgres: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	if err := testutil.RunMigrations(dbURL); err != nil {
		t.Fatalf("running migrations: %v", err)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// MinIO
	minioContainer, endpoint, err := testutil.MinIOContainer(ctx)
	if err != nil {
		t.Fatalf("starting minio: %v", err)
	}
	t.Cleanup(func() { _ = minioContainer.Terminate(ctx) })

	minioClient, err := storage.NewMinIOClient(endpoint, minioAccessKey, minioSecretKey, false)
	if err != nil {
		t.Fatalf("creating minio client: %v", err)
	}

	var bucketErr error
	for range 10 {
		if bucketErr = minioClient.EnsureBucket(ctx, minioBucket); bucketErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if bucketErr != nil {
		t.Fatalf("creating bucket: %v", bucketErr)
	}

	// Redis
	redisContainer, redisURL, err := testutil.RedisContainer(ctx)
	if err != nil {
		t.Fatalf("starting redis: %v", err)
	}
	t.Cleanup(func() { _ = redisContainer.Terminate(ctx) })

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("parsing redis URL: %v", err)
	}
	redisClient := redis.NewClient(opts)
	t.Cleanup(func() { _ = redisClient.Close() })

	return &integrationEnv{
		queries: store.New(db),
		minio:   minioClient,
		queue:   worker.NewRedisQueue(redisClient),
	}
}

// demoFileServer returns an httptest.Server that serves valid demo bytes at any path.
func demoFileServer(t *testing.T) *httptest.Server {
	t.Helper()
	body := make([]byte, 128)
	copy(body, demo.MagicCS2)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// createTestUser inserts a user into the DB and returns the user ID.
func createTestUser(t *testing.T, queries *store.Queries) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	user, err := queries.CreateUser(ctx, store.CreateUserParams{
		FaceitID: "integration-test-player-" + uuid.New().String()[:8],
		Nickname: "tester",
	})
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return user.ID
}

// createTestFaceitMatch inserts a faceit_match into the DB and returns it.
func createTestFaceitMatch(t *testing.T, queries *store.Queries, userID uuid.UUID, demoURL string) store.FaceitMatch {
	t.Helper()
	ctx := context.Background()

	match, err := queries.UpsertFaceitMatch(ctx, store.UpsertFaceitMatchParams{
		UserID:        userID,
		FaceitMatchID: "test-faceit-match-" + uuid.New().String()[:8],
		MapName:       "de_dust2",
		ScoreTeam:     16,
		ScoreOpponent: 10,
		Result:        "W",
		EloBefore:     sql.NullInt32{Int32: 2000, Valid: true},
		EloAfter:      sql.NullInt32{Int32: 2025, Valid: true},
		DemoUrl:       sql.NullString{String: demoURL, Valid: demoURL != ""},
		PlayedAt:      time.Now().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("creating test faceit match: %v", err)
	}
	return match
}

func TestImport_Integration_FullFlow(t *testing.T) {
	env := setupIntegrationEnv(t)
	ctx := context.Background()

	srv := demoFileServer(t)
	userID := createTestUser(t, env.queries)
	match := createTestFaceitMatch(t, env.queries, userID, srv.URL+"/demo.dem")

	importer := faceit.NewDemoImporter(
		env.queries, env.minio, env.queue,
		&http.Client{Timeout: 30 * time.Second}, minioBucket,
	)

	result, err := importer.Import(ctx, userID, match.ID, match.FaceitMatchID, match.DemoUrl.String, match.PlayedAt)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	if result.DemoID == uuid.Nil {
		t.Error("expected non-nil demo ID")
	}
	if result.FileSize != 128 {
		t.Errorf("file_size = %d, want 128", result.FileSize)
	}

	// Verify demo exists in DB
	demoRecord, err := env.queries.GetDemoByID(ctx, result.DemoID)
	if err != nil {
		t.Fatalf("getting demo by ID: %v", err)
	}
	if demoRecord.Status != "uploaded" {
		t.Errorf("demo status = %q, want uploaded", demoRecord.Status)
	}
	if !demoRecord.FaceitMatchID.Valid || demoRecord.FaceitMatchID.String != match.FaceitMatchID {
		t.Errorf("demo faceit_match_id = %v, want %s", demoRecord.FaceitMatchID, match.FaceitMatchID)
	}

	// Verify faceit_match.demo_id is linked
	updatedMatch, err := env.queries.GetFaceitMatchByID(ctx, match.ID)
	if err != nil {
		t.Fatalf("getting updated match: %v", err)
	}
	if !updatedMatch.DemoID.Valid || updatedMatch.DemoID.UUID != result.DemoID {
		t.Errorf("match demo_id = %v, want %s", updatedMatch.DemoID, result.DemoID)
	}

	// Verify MinIO object exists
	exists, err := env.minio.ObjectExists(ctx, minioBucket, demoRecord.FilePath)
	if err != nil {
		t.Fatalf("checking minio object: %v", err)
	}
	if !exists {
		t.Error("expected demo file to exist in MinIO")
	}
}

func TestImportByMatchID_Integration(t *testing.T) {
	env := setupIntegrationEnv(t)
	ctx := context.Background()

	srv := demoFileServer(t)
	userID := createTestUser(t, env.queries)
	match := createTestFaceitMatch(t, env.queries, userID, srv.URL+"/demo.dem")

	importer := faceit.NewDemoImporter(
		env.queries, env.minio, env.queue,
		&http.Client{Timeout: 30 * time.Second}, minioBucket,
	)

	result, err := importer.ImportByMatchID(ctx, userID, match.ID)
	if err != nil {
		t.Fatalf("ImportByMatchID failed: %v", err)
	}

	if result.DemoID == uuid.Nil {
		t.Error("expected non-nil demo ID")
	}

	// Verify demo_id is linked
	updatedMatch, err := env.queries.GetFaceitMatchByID(ctx, match.ID)
	if err != nil {
		t.Fatalf("getting updated match: %v", err)
	}
	if !updatedMatch.DemoID.Valid {
		t.Error("expected demo_id to be linked after ImportByMatchID")
	}

	// Calling again should return ErrDemoAlreadyLinked
	_, err = importer.ImportByMatchID(ctx, userID, match.ID)
	if !errors.Is(err, faceit.ErrDemoAlreadyLinked) {
		t.Errorf("second import error = %v, want ErrDemoAlreadyLinked", err)
	}
}

func TestImport_Integration_DownloadFailureDoesNotBlockSync(t *testing.T) {
	env := setupIntegrationEnv(t)
	ctx := context.Background()

	// Server that 500s for /fail and 200s for /ok
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body := make([]byte, 128)
		copy(body, demo.MagicCS2)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)

	userID := createTestUser(t, env.queries)
	failMatch := createTestFaceitMatch(t, env.queries, userID, srv.URL+"/fail")
	okMatch := createTestFaceitMatch(t, env.queries, userID, srv.URL+"/ok")

	importer := faceit.NewDemoImporter(
		env.queries, env.minio, env.queue,
		&http.Client{Timeout: 30 * time.Second}, minioBucket,
	)

	// Import the failing one — should return error
	_, err := importer.Import(ctx, userID, failMatch.ID, failMatch.FaceitMatchID, failMatch.DemoUrl.String, failMatch.PlayedAt)
	if err == nil {
		t.Error("expected error for failed download")
	}

	// Import the working one — should succeed
	result, err := importer.Import(ctx, userID, okMatch.ID, okMatch.FaceitMatchID, okMatch.DemoUrl.String, okMatch.PlayedAt)
	if err != nil {
		t.Fatalf("import of ok match failed: %v", err)
	}
	if result.DemoID == uuid.Nil {
		t.Error("expected non-nil demo ID for ok match")
	}

	// Verify the failing match has no demo linked
	failUpdated, err := env.queries.GetFaceitMatchByID(ctx, failMatch.ID)
	if err != nil {
		t.Fatalf("getting fail match: %v", err)
	}
	if failUpdated.DemoID.Valid {
		t.Error("failing match should not have demo_id linked")
	}

	// Verify the ok match does have demo linked
	okUpdated, err := env.queries.GetFaceitMatchByID(ctx, okMatch.ID)
	if err != nil {
		t.Fatalf("getting ok match: %v", err)
	}
	if !okUpdated.DemoID.Valid {
		t.Error("ok match should have demo_id linked")
	}
}
