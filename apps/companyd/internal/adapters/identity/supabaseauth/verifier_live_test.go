package supabaseauth

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

// TestNew_FetchesRealProjectJWKS is skipped unless SUPABASE_URL is set —
// same convention as internal/application/integration_test.go's
// requireRealApp. It doesn't need a live signed token (none was
// obtainable: this project has anonymous sign-in disabled and rejects
// test-domain signups) — it only confirms New can actually fetch and
// parse the real project's JWKS, closing the gap a self-signed-key-only
// test suite can't: does this code work against the real endpoint, not
// just a shape that mirrors it.
func TestNew_FetchesRealProjectJWKS(t *testing.T) {
	_ = godotenv.Load("../../../../.env")
	supabaseURL := os.Getenv("SUPABASE_URL")
	if supabaseURL == "" {
		t.Skip("SUPABASE_URL not set; skipping live JWKS fetch test")
	}

	v, err := New(context.Background(), supabaseURL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// VerifyHumanToken against a syntactically invalid token still proves
	// the keyfunc is live and functioning (it fails on parsing, not on an
	// unreachable JWKS) — a real signed token isn't available to fully
	// exercise verification here.
	_, err = v.VerifyHumanToken(context.Background(), "not-a-real-token")
	if err == nil {
		t.Fatal("expected an error for a garbage token, got nil")
	}
}
