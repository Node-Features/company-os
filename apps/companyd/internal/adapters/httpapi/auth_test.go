package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
)

// fakeAuthenticator implements ports.Authenticator for auth_test.go only.
type fakeAuthenticator struct {
	evidence principal.AuthenticatedEvidence
	err      error
}

func (f fakeAuthenticator) VerifyHumanToken(ctx context.Context, rawToken string) (principal.AuthenticatedEvidence, error) {
	return f.evidence, f.err
}

func TestRequireHumanAuth(t *testing.T) {
	validEvidence := principal.AuthenticatedEvidence{Subject: "user-123", PrincipalType: principal.KindHuman}

	tests := []struct {
		name       string
		authHeader string
		authn      ports.Authenticator
		wantStatus int
		wantCalled bool
	}{
		{
			name:       "missing header",
			authHeader: "",
			authn:      fakeAuthenticator{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed header, no Bearer prefix",
			authHeader: "Token abc",
			authn:      fakeAuthenticator{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "empty bearer token",
			authHeader: "Bearer ",
			authn:      fakeAuthenticator{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "expired token",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrExpiredToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong issuer",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrWrongIssuer},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "wrong audience",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrWrongAudience},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid signature",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrInvalidSignature},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed token",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrMalformedToken},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "verifier unavailable",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{err: ports.ErrVerifierUnavailable},
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "valid token",
			authHeader: "Bearer abc",
			authn:      fakeAuthenticator{evidence: validEvidence},
			wantStatus: http.StatusOK,
			wantCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotEvidence principal.AuthenticatedEvidence
			var gotOK bool

			next := func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotEvidence, gotOK = EvidenceFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}

			req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			RequireHumanAuth(tt.authn, next)(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tt.wantCalled)
			}
			if tt.wantCalled {
				if !gotOK {
					t.Fatal("expected evidence in context, found none")
				}
				if gotEvidence.Subject != validEvidence.Subject {
					t.Fatalf("evidence.Subject = %q, want %q", gotEvidence.Subject, validEvidence.Subject)
				}
			}
		})
	}
}
