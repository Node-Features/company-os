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

// pendingApprovalView is the Approval inbox's list response shape — omits
// DecidedByPrincipalID/DecidedAt/Reason, which are always nil on a PENDING
// row and don't belong in this response.
type pendingApprovalView struct {
	ApprovalID            string `json:"approvalId"`
	Action                string `json:"action"`
	ResourceType          string `json:"resourceType"`
	ResourceID            string `json:"resourceId"`
	RequestingPrincipalID string `json:"requestingPrincipalId"`
	CreatedAt             string `json:"createdAt"`
}

// ListPendingApprovalsHandler handles GET /v1/approvals?status=PENDING —
// the Approval inbox (ROADMAP.md Phase 10 Slice 1). status is required and
// must be exactly PENDING this slice; every other Approval status is
// already terminal/consumed and isn't what an "inbox" means.
func ListPendingApprovalsHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if status := r.URL.Query().Get("status"); status != "PENDING" {
			writeError(w, http.StatusBadRequest, "status must be PENDING")
			return
		}
		approvals, err := app.ListPendingApprovals(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		views := make([]pendingApprovalView, len(approvals))
		for i, a := range approvals {
			views[i] = pendingApprovalView{
				ApprovalID:            a.ApprovalID.String(),
				Action:                a.Action,
				ResourceType:          a.ResourceType,
				ResourceID:            a.ResourceID,
				RequestingPrincipalID: a.RequestingPrincipalID.String(),
				CreatedAt:             a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(views)
	}
}
