package supabaseauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/golang-jwt/jwt/v5"
)

const testKID = "test-key-1"

// testJWKSServer serves priv's public key as a JWKS in the exact shape
// confirmed live against the real Supabase project this session
// (curl "${SUPABASE_URL}/auth/v1/.well-known/jwks.json"): one EC/P-256
// ES256 signing key.
func testJWKSServer(t *testing.T, priv *ecdsa.PrivateKey) *httptest.Server {
	t.Helper()
	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	xBytes := priv.PublicKey.X.FillBytes(make([]byte, 32))
	yBytes := priv.PublicKey.Y.FillBytes(make([]byte, 32))

	body, err := json.Marshal(map[string]any{
		"keys": []map[string]any{
			{
				"kty":     "EC",
				"crv":     "P-256",
				"alg":     "ES256",
				"use":     "sig",
				"key_ops": []string{"verify"},
				"kid":     testKID,
				"x":       enc(xBytes),
				"y":       enc(yBytes),
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal JWKS: %v", err)
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))
}

// newTestVerifier builds a Verifier whose JWKS is served locally by a
// server mirroring the real project's shape, rather than depending on
// Supabase — construction mirrors New, but points issuer/jwksURL at the
// test server instead of the "{supabaseURL}/auth/v1" pattern New derives.
func newTestVerifier(t *testing.T, jwksServerURL string) *Verifier {
	t.Helper()
	v, err := newWithJWKSURL(context.Background(), "https://test-project.supabase.co/auth/v1", jwksServerURL)
	if err != nil {
		t.Fatalf("newWithJWKSURL: %v", err)
	}
	return v
}

func signTestToken(t *testing.T, priv *ecdsa.PrivateKey, mutate func(*claims)) string {
	t.Helper()
	now := time.Now()
	c := claims{
		Email: "human@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://test-project.supabase.co/auth/v1",
			Subject:   "5b1b2c3d-0000-4000-8000-000000000099",
			Audience:  jwt.ClaimStrings{authenticatedAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
	if mutate != nil {
		mutate(&c)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
	token.Header["kid"] = testKID
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}

func TestVerifyHumanToken_ValidTokenProducesCorrectEvidence(t *testing.T) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	tok := signTestToken(t, priv, nil)
	evidence, err := v.VerifyHumanToken(context.Background(), tok)
	if err != nil {
		t.Fatalf("VerifyHumanToken: %v", err)
	}
	if evidence.PrincipalType != principal.KindHuman {
		t.Errorf("PrincipalType = %s, want HUMAN", evidence.PrincipalType)
	}
	if evidence.Subject != "5b1b2c3d-0000-4000-8000-000000000099" {
		t.Errorf("Subject = %q, unexpected", evidence.Subject)
	}
	if evidence.Email == nil || *evidence.Email != "human@example.com" {
		t.Errorf("Email = %v, want human@example.com", evidence.Email)
	}
	if evidence.Audience != authenticatedAudience {
		t.Errorf("Audience = %q, want %q", evidence.Audience, authenticatedAudience)
	}
	if evidence.VerifierAdapter != verifierAdapterName {
		t.Errorf("VerifierAdapter = %q, want %q", evidence.VerifierAdapter, verifierAdapterName)
	}
	if evidence.ExpiresAt.Before(evidence.IssuedAt) {
		t.Errorf("ExpiresAt %v is before IssuedAt %v", evidence.ExpiresAt, evidence.IssuedAt)
	}
}

func TestVerifyHumanToken_Expired(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	tok := signTestToken(t, priv, func(c *claims) {
		past := time.Now().Add(-2 * time.Hour)
		c.IssuedAt = jwt.NewNumericDate(past)
		c.ExpiresAt = jwt.NewNumericDate(past.Add(time.Hour))
	})
	_, err := v.VerifyHumanToken(context.Background(), tok)
	if !errors.Is(err, ports.ErrExpiredToken) {
		t.Fatalf("err = %v, want wrapping ports.ErrExpiredToken", err)
	}
}

func TestVerifyHumanToken_WrongIssuer(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	tok := signTestToken(t, priv, func(c *claims) {
		c.Issuer = "https://attacker.example.com/auth/v1"
	})
	_, err := v.VerifyHumanToken(context.Background(), tok)
	if !errors.Is(err, ports.ErrWrongIssuer) {
		t.Fatalf("err = %v, want wrapping ports.ErrWrongIssuer", err)
	}
}

func TestVerifyHumanToken_WrongAudience(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	tok := signTestToken(t, priv, func(c *claims) {
		c.Audience = jwt.ClaimStrings{"anon"}
	})
	_, err := v.VerifyHumanToken(context.Background(), tok)
	if !errors.Is(err, ports.ErrWrongAudience) {
		t.Fatalf("err = %v, want wrapping ports.ErrWrongAudience", err)
	}
}

func TestVerifyHumanToken_UnknownKID(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	now := time.Now()
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "https://test-project.supabase.co/auth/v1", Subject: "u1",
		Audience: jwt.ClaimStrings{authenticatedAudience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, c)
	token.Header["kid"] = "some-other-kid-never-published"
	tok, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	_, err = v.VerifyHumanToken(context.Background(), tok)
	if err == nil {
		t.Fatal("expected an error for an unknown kid, got nil")
	}
}

func TestVerifyHumanToken_TamperedSignature(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	tok := signTestToken(t, priv, nil)
	tampered := tok[:len(tok)-4] + "AAAA" // corrupt the signature segment
	_, err := v.VerifyHumanToken(context.Background(), tampered)
	if err == nil {
		t.Fatal("expected an error for a tampered signature, got nil")
	}
}

// TestVerifyHumanToken_WrongAlgorithmRejected proves the alg-confusion
// guard (jwt.WithValidMethods) holds: a token claiming HS256 and "signed"
// with the ES256 public key's coordinates as an HMAC secret (the classic
// RS/ES-to-HS confusion attack shape) must never verify.
func TestVerifyHumanToken_WrongAlgorithmRejected(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	now := time.Now()
	c := claims{RegisteredClaims: jwt.RegisteredClaims{
		Issuer: "https://test-project.supabase.co/auth/v1", Subject: "u1",
		Audience: jwt.ClaimStrings{authenticatedAudience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	token.Header["kid"] = testKID
	tok, err := token.SignedString(priv.PublicKey.X.Bytes())
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	_, err = v.VerifyHumanToken(context.Background(), tok)
	if err == nil {
		t.Fatal("expected an error for a token using a non-whitelisted algorithm, got nil")
	}
}

func TestVerifyHumanToken_MalformedTokenRejected(t *testing.T) {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	server := testJWKSServer(t, priv)
	defer server.Close()
	v := newTestVerifier(t, server.URL)

	_, err := v.VerifyHumanToken(context.Background(), "not-a-jwt-at-all")
	if !errors.Is(err, ports.ErrMalformedToken) {
		t.Fatalf("err = %v, want wrapping ports.ErrMalformedToken", err)
	}
}
