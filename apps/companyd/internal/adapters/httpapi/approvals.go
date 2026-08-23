package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/google/uuid"
)

// ResolveApprovalHandler handles POST /v1/approvals/{approvalId}/decide.
// The deciding Principal is never read from the request — Application
// always resolves it to fixtures.Registry.ApproverPrincipal(), the same
// never-client-asserted-identity pattern every other endpoint follows this
// slice.
func ResolveApprovalHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			ApprovalID: approvalID,
			Approve:    body.Approve,
			Reason:     body.Reason,
		})
		writeResult(w, uuid.New(), res)
	}
}
