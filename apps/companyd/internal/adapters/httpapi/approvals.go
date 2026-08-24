package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/google/uuid"
)

// ResolveApprovalHandler handles POST /v1/approvals/{approvalId}/decide.
// The deciding Principal is the real, context-resolved caller
// (docs/audit/gap-approval-principal-attribution.md, fixed 2026-08-25) —
// never client-asserted, same as every other endpoint's principal.
func ResolveApprovalHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decidingPrincipal, ok := PrincipalFromContext(r.Context())
		if !ok {
			// Programmer error, not a caller fault: this route is only ever
			// wired behind RequireHumanAuth+ResolvePrincipal.
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		approvalID, err := uuid.Parse(r.PathValue("approvalId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid approvalId")
			return
		}
		var body struct {
			Approve bool    `json:"approve"`
			Reason  *string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		res := app.ResolveApproval(r.Context(), application.ResolveApprovalRequest{
			ApprovalID:        approvalID,
			Approve:           body.Approve,
			Reason:            body.Reason,
			DecidingPrincipal: decidingPrincipal,
		})
		writeResult(w, uuid.New(), res)
	}
}
