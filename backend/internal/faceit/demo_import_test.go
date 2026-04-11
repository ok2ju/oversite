package faceit_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ok2ju/oversite/backend/internal/demo"
	"github.com/ok2ju/oversite/backend/internal/faceit"
	"github.com/ok2ju/oversite/backend/internal/store"
)

// --- Mocks ---

type mockImportStore struct {
	getMatchResult store.FaceitMatch
	getMatchErr    error

	createDemoResult store.Demo
	createDemoErr    error
	createDemoCalls  []store.CreateDemoParams

	linkResult store.FaceitMatch
	linkErr    error
	linkCalls  []store.LinkFaceitMatchToDemoParams
}

func (m *mockImportStore) GetFaceitMatchByID(_ context.Context, id uuid.UUID) (store.FaceitMatch, error) {
	if m.getMatchErr != nil {
		return store.FaceitMatch{}, m.getMatchErr
	}
	return m.getMatchResult, nil
}

func (m *mockImportStore) CreateDemo(_ context.Context, arg store.CreateDemoParams) (store.Demo, error) {
	m.createDemoCalls = append(m.createDemoCalls, arg)
	if m.createDemoErr != nil {
		return store.Demo{}, m.createDemoErr
	}
	return m.createDemoResult, nil
}

func (m *mockImportStore) LinkFaceitMatchToDemo(_ context.Context, arg store.LinkFaceitMatchToDemoParams) (store.FaceitMatch, error) {
	m.linkCalls = append(m.linkCalls, arg)
	if m.linkErr != nil {
		return store.FaceitMatch{}, m.linkErr
	}
	return m.linkResult, nil
}

type mockImportS3 struct {
	putErr      error
	putCalls    []string // keys
	deleteErr   error
	deleteCalls []string // keys
}

func (m *mockImportS3) PutObject(_ context.Context, _, key string, _ io.Reader, _ int64) error {
	m.putCalls = append(m.putCalls, key)
	if m.putErr != nil {
		return m.putErr
	}
	return nil
}

func (m *mockImportS3) DeleteObject(_ context.Context, _, key string) error {
	m.deleteCalls = append(m.deleteCalls, key)
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

type mockImportQueue struct {
	err   error
	calls []string // streams
}

func (m *mockImportQueue) Enqueue(_ context.Context, stream string, _ map[string]interface{}) (string, error) {
	m.calls = append(m.calls, stream)
	if m.err != nil {
		return "", m.err
	}
	return "msg-1", nil
}

type mockHTTPDownloader struct {
	resp *http.Response
	err  error
}

func (m *mockHTTPDownloader) Do(_ *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

// --- Helpers ---

func validDemoBytes() []byte {
	// CS2 magic bytes + some padding to make a valid-looking file
	buf := make([]byte, 64)
	copy(buf, demo.MagicCS2)
	return buf
}

func httpResponseWithBody(statusCode int, body []byte) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

var (
	importUserID  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	importMatchID = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	importDemoID  = uuid.MustParse("44444444-4444-4444-4444-444444444444")
)

func newTestImporter(st *mockImportStore, s3 *mockImportS3, q *mockImportQueue, dl *mockHTTPDownloader) *faceit.DemoImporter {
	return faceit.NewDemoImporter(st, s3, q, dl, "test-bucket")
}

// --- Import Tests ---

func TestImport_HappyPath(t *testing.T) {
	st := &mockImportStore{
		createDemoResult: store.Demo{ID: importDemoID, Status: "uploaded", FileSize: 64},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	result, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DemoID == uuid.Nil {
		t.Error("expected non-nil demo ID")
	}
	if len(s3.putCalls) != 1 {
		t.Errorf("S3 PutObject called %d times, want 1", len(s3.putCalls))
	}
	if len(st.createDemoCalls) != 1 {
		t.Errorf("CreateDemo called %d times, want 1", len(st.createDemoCalls))
	}
	if st.createDemoCalls[0].Status != "uploaded" {
		t.Errorf("demo status = %q, want uploaded", st.createDemoCalls[0].Status)
	}
	if len(st.linkCalls) != 1 {
		t.Errorf("LinkFaceitMatchToDemo called %d times, want 1", len(st.linkCalls))
	}
	if len(q.calls) != 1 {
		t.Errorf("Enqueue called %d times, want 1", len(q.calls))
	}
	if q.calls[0] != "demo_parse" {
		t.Errorf("enqueue stream = %q, want demo_parse", q.calls[0])
	}
}

func TestImport_DownloadHTTPError(t *testing.T) {
	st := &mockImportStore{}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{err: errors.New("connection refused")}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if !errors.Is(err, faceit.ErrDownloadFailed) {
		t.Errorf("error = %v, want ErrDownloadFailed", err)
	}
}

func TestImport_DownloadNon200(t *testing.T) {
	st := &mockImportStore{}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusNotFound, []byte("not found"))}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if !errors.Is(err, faceit.ErrDownloadFailed) {
		t.Errorf("error = %v, want ErrDownloadFailed", err)
	}
}

func TestImport_InvalidMagicBytes(t *testing.T) {
	badBytes := make([]byte, 64)
	copy(badBytes, []byte("NOTADEMO"))

	st := &mockImportStore{}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, badBytes)}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if !errors.Is(err, faceit.ErrInvalidDemo) {
		t.Errorf("error = %v, want ErrInvalidDemo", err)
	}
	if len(s3.putCalls) != 0 {
		t.Error("S3 PutObject should not be called for invalid demo")
	}
}

func TestImport_S3UploadFails(t *testing.T) {
	st := &mockImportStore{}
	s3 := &mockImportS3{putErr: errors.New("s3 down")}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(st.createDemoCalls) != 0 {
		t.Error("CreateDemo should not be called when S3 upload fails")
	}
}

func TestImport_CreateDemoFails_CleansUpS3(t *testing.T) {
	st := &mockImportStore{createDemoErr: errors.New("db constraint")}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if len(s3.putCalls) != 1 {
		t.Errorf("S3 PutObject should be called once, got %d", len(s3.putCalls))
	}
	if len(s3.deleteCalls) != 1 {
		t.Errorf("S3 DeleteObject should be called once for cleanup, got %d", len(s3.deleteCalls))
	}
}

func TestImport_LinkFails_CleansUpS3(t *testing.T) {
	st := &mockImportStore{
		createDemoResult: store.Demo{ID: importDemoID, Status: "uploaded", FileSize: 64},
		linkErr:          errors.New("fk violation"),
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if err == nil {
		t.Fatal("expected error from link failure")
	}
	// Demo was still created
	if len(st.createDemoCalls) != 1 {
		t.Error("CreateDemo should still be called")
	}
	// S3 object should be uploaded then cleaned up
	if len(s3.putCalls) != 1 {
		t.Errorf("S3 PutObject should be called once, got %d", len(s3.putCalls))
	}
	if len(s3.deleteCalls) != 1 {
		t.Errorf("S3 DeleteObject should be called once for cleanup, got %d", len(s3.deleteCalls))
	}
}

func TestImport_EnqueueFails_NonFatal(t *testing.T) {
	st := &mockImportStore{
		createDemoResult: store.Demo{ID: importDemoID, Status: "uploaded", FileSize: 64},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{err: errors.New("redis down")}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	result, err := imp.Import(context.Background(), importUserID, importMatchID, "faceit-match-1", "https://demo.faceit.com/1.dem", time.Now())
	if err != nil {
		t.Fatalf("enqueue failure should be non-fatal, got: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// --- ImportByMatchID Tests ---

func TestImportByMatchID_HappyPath(t *testing.T) {
	st := &mockImportStore{
		getMatchResult: store.FaceitMatch{
			ID:            importMatchID,
			UserID:        importUserID,
			FaceitMatchID: "faceit-match-1",
			DemoUrl:       sql.NullString{String: "https://demo.faceit.com/1.dem", Valid: true},
			PlayedAt:      time.Now(),
		},
		createDemoResult: store.Demo{ID: importDemoID, Status: "uploaded", FileSize: 64},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{resp: httpResponseWithBody(http.StatusOK, validDemoBytes())}

	imp := newTestImporter(st, s3, q, dl)
	result, err := imp.ImportByMatchID(context.Background(), importUserID, importMatchID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestImportByMatchID_NotFound(t *testing.T) {
	st := &mockImportStore{getMatchErr: sql.ErrNoRows}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.ImportByMatchID(context.Background(), importUserID, importMatchID)
	if !errors.Is(err, faceit.ErrMatchNotFound) {
		t.Errorf("error = %v, want ErrMatchNotFound", err)
	}
}

func TestImportByMatchID_WrongUser(t *testing.T) {
	otherUser := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	st := &mockImportStore{
		getMatchResult: store.FaceitMatch{
			ID:     importMatchID,
			UserID: otherUser,
		},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.ImportByMatchID(context.Background(), importUserID, importMatchID)
	if !errors.Is(err, faceit.ErrMatchForbidden) {
		t.Errorf("error = %v, want ErrMatchForbidden", err)
	}
}

func TestImportByMatchID_NoDemoURL(t *testing.T) {
	st := &mockImportStore{
		getMatchResult: store.FaceitMatch{
			ID:      importMatchID,
			UserID:  importUserID,
			DemoUrl: sql.NullString{Valid: false},
		},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.ImportByMatchID(context.Background(), importUserID, importMatchID)
	if !errors.Is(err, faceit.ErrNoDemoURL) {
		t.Errorf("error = %v, want ErrNoDemoURL", err)
	}
}

func TestImportByMatchID_AlreadyLinked(t *testing.T) {
	st := &mockImportStore{
		getMatchResult: store.FaceitMatch{
			ID:      importMatchID,
			UserID:  importUserID,
			DemoUrl: sql.NullString{String: "https://demo.faceit.com/1.dem", Valid: true},
			DemoID:  uuid.NullUUID{UUID: importDemoID, Valid: true},
		},
	}
	s3 := &mockImportS3{}
	q := &mockImportQueue{}
	dl := &mockHTTPDownloader{}

	imp := newTestImporter(st, s3, q, dl)
	_, err := imp.ImportByMatchID(context.Background(), importUserID, importMatchID)
	if !errors.Is(err, faceit.ErrDemoAlreadyLinked) {
		t.Errorf("error = %v, want ErrDemoAlreadyLinked", err)
	}
}
