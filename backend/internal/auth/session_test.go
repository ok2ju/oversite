package auth_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/ok2ju/oversite/backend/internal/auth"
)

func TestGenerateSessionToken_Length(t *testing.T) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}
}

func TestGenerateSessionToken_HexCharset(t *testing.T) {
	token, err := auth.GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := hex.DecodeString(token); err != nil {
		t.Errorf("token is not valid hex: %q", token)
	}
}

func TestGenerateSessionToken_Uniqueness(t *testing.T) {
	t1, _ := auth.GenerateSessionToken()
	t2, _ := auth.GenerateSessionToken()
	if t1 == t2 {
		t.Error("two tokens should differ")
	}
}

func TestGenerateSessionTokenWithReader_Deterministic(t *testing.T) {
	input := bytes.Repeat([]byte{0xAB}, 32)
	reader := bytes.NewReader(input)

	token, err := auth.GenerateSessionTokenWithReader(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := hex.EncodeToString(input)
	if token != expected {
		t.Errorf("expected %q, got %q", expected, token)
	}
}

func TestSessionData_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	original := &auth.SessionData{
		UserID:       "user-123",
		FaceitID:     "faceit-456",
		Nickname:     "player1",
		AccessToken:  "access-tok",
		RefreshToken: "refresh-tok",
		CreatedAt:    now,
		ExpiresAt:    now.Add(7 * 24 * time.Hour),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var restored auth.SessionData
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"UserID", restored.UserID, original.UserID},
		{"FaceitID", restored.FaceitID, original.FaceitID},
		{"Nickname", restored.Nickname, original.Nickname},
		{"AccessToken", restored.AccessToken, original.AccessToken},
		{"RefreshToken", restored.RefreshToken, original.RefreshToken},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}

	if !restored.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", restored.CreatedAt, original.CreatedAt)
	}
	if !restored.ExpiresAt.Equal(original.ExpiresAt) {
		t.Errorf("ExpiresAt: got %v, want %v", restored.ExpiresAt, original.ExpiresAt)
	}
}

func TestSessionData_UnmarshalInvalidJSON(t *testing.T) {
	var data auth.SessionData
	err := json.Unmarshal([]byte(`{invalid`), &data)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
