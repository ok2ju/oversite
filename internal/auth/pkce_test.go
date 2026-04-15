package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"regexp"
	"testing"
)

func TestGenerateCodeVerifier(t *testing.T) {
	tests := []struct {
		name  string
		check func(t *testing.T, v string)
	}{
		{
			name: "length is 43 for 32 bytes base64url",
			check: func(t *testing.T, v string) {
				t.Helper()
				if len(v) != 43 {
					t.Errorf("verifier length = %d, want 43", len(v))
				}
			},
		},
		{
			name: "contains only base64url characters",
			check: func(t *testing.T, v string) {
				t.Helper()
				re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
				if !re.MatchString(v) {
					t.Errorf("verifier contains invalid characters: %q", v)
				}
			},
		},
		{
			name: "no padding characters",
			check: func(t *testing.T, v string) {
				t.Helper()
				for _, c := range v {
					if c == '=' {
						t.Errorf("verifier should not contain '=': %q", v)
						return
					}
				}
			},
		},
		{
			name: "no standard base64 characters + and /",
			check: func(t *testing.T, v string) {
				t.Helper()
				for _, c := range v {
					if c == '+' || c == '/' {
						t.Errorf("verifier should not contain '+' or '/': %q", v)
						return
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := GenerateCodeVerifier()
			if err != nil {
				t.Fatalf("GenerateCodeVerifier: %v", err)
			}
			tt.check(t, v)
		})
	}
}

func TestGenerateCodeVerifier_Uniqueness(t *testing.T) {
	v1, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	v2, err := GenerateCodeVerifier()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if v1 == v2 {
		t.Error("two verifiers should differ")
	}
}

func TestGenerateCodeVerifierWithReader_Deterministic(t *testing.T) {
	input := bytes.Repeat([]byte{0xAB}, 32)
	reader := bytes.NewReader(input)

	v, err := GenerateCodeVerifierWithReader(reader)
	if err != nil {
		t.Fatalf("GenerateCodeVerifierWithReader: %v", err)
	}

	want := base64.RawURLEncoding.EncodeToString(input)
	if v != want {
		t.Errorf("verifier = %q, want %q", v, want)
	}
}

func TestGenerateCodeVerifierWithReader_ShortRead(t *testing.T) {
	reader := bytes.NewReader([]byte{0x01, 0x02}) // only 2 bytes, need 32
	_, err := GenerateCodeVerifierWithReader(reader)
	if err == nil {
		t.Fatal("expected error for short reader, got nil")
	}
}

func TestGenerateCodeChallenge(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
	}{
		{
			name:     "RFC 7636 test vector",
			verifier: "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
		},
		{
			name:     "simple verifier",
			verifier: "test-verifier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateCodeChallenge(tt.verifier)

			// Recompute expected value.
			h := sha256.Sum256([]byte(tt.verifier))
			want := base64.RawURLEncoding.EncodeToString(h[:])

			if got != want {
				t.Errorf("challenge = %q, want %q", got, want)
			}
		})
	}
}

func TestGenerateCodeChallenge_NoPadding(t *testing.T) {
	challenge := GenerateCodeChallenge("test-verifier")
	for _, c := range challenge {
		if c == '=' {
			t.Errorf("challenge should not contain padding: %q", challenge)
			return
		}
	}
}

func TestGenerateCodeChallenge_NoStandardBase64Chars(t *testing.T) {
	challenge := GenerateCodeChallenge("test-verifier")
	for _, c := range challenge {
		if c == '+' || c == '/' {
			t.Errorf("challenge should not contain '+' or '/': %q", challenge)
			return
		}
	}
}
