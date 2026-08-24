package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/Node-Features/company-os/apps/companyd/internal/identity"
	"github.com/google/uuid"
)

// fakePrincipalRepository implements ports.PrincipalRepository for
// principal_test.go only.
type fakePrincipalRepository struct {
	principal principal.Principal
	err       error
}

func (f fakePrincipalRepository) FindOrCreateHumanBinding(ctx context.Context, organizationID uuid.UUID, evidence principal.AuthenticatedEvidence) (principal.Principal, error) {
	return f.principal, f.err
}

func TestResolvePrincipal(t *testing.T) {
	resolved := principal.Principal{PrincipalID: uuid.New(), Kind: principal.KindHuman, DisplayName: "test@example.com"}
	evidence := principal.AuthenticatedEvidence{Subject: "user-123", PrincipalType: principal.KindHuman}

	tests := []struct {
		name          string
		priorEvidence bool
		repo          fakePrincipalRepository
		wantStatus    int
		wantCalled    bool
		wantPrincipal bool
	}{
		{
			name:          "no prior evidence in context (RequireHumanAuth not run first)",
			priorEvidence: false,
			wantStatus:    http.StatusInternalServerError,
		},
		{
			name:          "resolver error",
			priorEvidence: true,
			repo:          fakePrincipalRepository{err: assertErr},
			wantStatus:    http.StatusServiceUnavailable,
		},
		{
			name:          "success",
			priorEvidence: true,
			repo:          fakePrincipalRepository{principal: resolved},
			wantStatus:    http.StatusOK,
			wantCalled:    true,
			wantPrincipal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotPrincipal principal.Principal
			var gotOK bool

			next := func(w http.ResponseWriter, r *http.Request) {
				called = true
				gotPrincipal, gotOK = PrincipalFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}

			req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
			if tt.priorEvidence {
				ctx := context.WithValue(req.Context(), evidenceContextKey{}, evidence)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			resolver := identity.NewResolver(tt.repo, uuid.New())
			ResolvePrincipal(resolver, next)(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantCalled {
				t.Fatalf("next called = %v, want %v", called, tt.wantCalled)
			}
			if tt.wantPrincipal {
				if !gotOK {
					t.Fatal("expected principal in context, found none")
				}
				if gotPrincipal.PrincipalID != resolved.PrincipalID {
					t.Fatalf("principal.PrincipalID = %v, want %v", gotPrincipal.PrincipalID, resolved.PrincipalID)
				}
			}
		})
	}
}

var assertErr = &testError{"resolver unavailable"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
