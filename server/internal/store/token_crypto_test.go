package store

import (
	"strings"
	"testing"
)

func TestTokenCrypto(t *testing.T) {
	t.Setenv("ATLAS_ENCRYPTION_KEY", "test-key-123")

	for _, tok := range []string{"secret123", "", "a long token w/ spaces & symbols !@#=/"} {
		enc := encToken(tok)
		if tok == "" {
			if enc != "" {
				t.Fatalf("empty token should stay empty, got %q", enc)
			}
			continue
		}
		if enc == tok || !strings.HasPrefix(enc, tokenEncPrefix) {
			t.Fatalf("token not encrypted: %q -> %q", tok, enc)
		}
		if got := decToken(enc); got != tok {
			t.Fatalf("roundtrip mismatch: got %q want %q", got, tok)
		}
	}

	// Legacy plaintext (no prefix) passes through unchanged.
	if got := decToken("legacy-plain"); got != "legacy-plain" {
		t.Fatalf("legacy passthrough failed: %q", got)
	}

	// Wrong key → graceful empty (caller surfaces the auth error).
	enc := encToken("secret")
	t.Setenv("ATLAS_ENCRYPTION_KEY", "a-different-key")
	if got := decToken(enc); got != "" {
		t.Fatalf("decrypt with wrong key should yield empty, got %q", got)
	}
}
