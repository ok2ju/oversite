package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// GenerateCodeVerifier generates a cryptographically random PKCE code verifier.
// The result is 32 random bytes, base64url-no-pad encoded (43 characters).
func GenerateCodeVerifier() (string, error) {
	return GenerateCodeVerifierWithReader(rand.Reader)
}

// GenerateCodeVerifierWithReader generates a PKCE code verifier from the given
// reader. This allows deterministic testing by supplying a fixed reader.
func GenerateCodeVerifierWithReader(r io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("generating code verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateCodeChallenge computes the S256 PKCE code challenge for a verifier.
// The result is SHA-256 of the verifier, base64url-no-pad encoded.
func GenerateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
