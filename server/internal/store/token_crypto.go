package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"strings"
)

// Source tokens are stored encrypted at rest (AES-256-GCM). The key is derived
// from ATLAS_ENCRYPTION_KEY, falling back to ATLAS_SESSION_SECRET — both stable
// across restarts. Set a dedicated ATLAS_ENCRYPTION_KEY to decouple token
// encryption from session-secret rotation. Stored form: "enc:" + base64(nonce‖ct).
const tokenEncPrefix = "enc:"

func tokenKey() []byte {
	s := os.Getenv("ATLAS_ENCRYPTION_KEY")
	if s == "" {
		s = os.Getenv("ATLAS_SESSION_SECRET")
	}
	if s == "" {
		s = "dev-insecure-change-me" // mirrors the config default
	}
	h := sha256.Sum256([]byte("atlas-token-v1:" + s)) // domain-separated from other uses
	return h[:]
}

// encToken encrypts a token for storage. Empty stays empty.
func encToken(plain string) string {
	if plain == "" {
		return ""
	}
	gcm, err := tokenGCM()
	if err != nil {
		return plain // never block on a crypto setup error
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return plain
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil) // nonce ‖ ciphertext
	return tokenEncPrefix + base64.StdEncoding.EncodeToString(sealed)
}

// decToken decrypts a stored token. Values without the "enc:" prefix are returned
// as-is (legacy plaintext, or empty). A decryption failure yields "" (the caller
// then syncs with no token and surfaces the auth error).
func decToken(stored string) string {
	if !strings.HasPrefix(stored, tokenEncPrefix) {
		return stored
	}
	raw, err := base64.StdEncoding.DecodeString(stored[len(tokenEncPrefix):])
	if err != nil {
		return ""
	}
	gcm, err := tokenGCM()
	if err != nil || len(raw) < gcm.NonceSize() {
		return ""
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return ""
	}
	return string(pt)
}

func tokenGCM() (cipher.AEAD, error) {
	block, err := aes.NewCipher(tokenKey())
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
