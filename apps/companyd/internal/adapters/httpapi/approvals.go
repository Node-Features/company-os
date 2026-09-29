package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/approval"
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

// ListApprovalsHandler handles GET /v1/approvals — Phase 10 Slice 1's
// list-pending endpoint. Query params: status (comma-separated, e.g.
// "PENDING,REJECTED"; omitted defaults to PENDING-only, per
// application.ListApprovals's doc comment), limit, offset (real
// page-based pagination — the UI pages through offset+limit, never
// fetches an unbounded list).
func ListApprovalsHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statuses []approval.Status
		if raw := r.URL.Query().Get("status"); raw != "" {
			for _, s := range strings.Split(raw, ",") {
				statuses = append(statuses, approval.Status(strings.TrimSpace(s)))
			}
		}
		limit := 0
		if raw := r.URL.Query().Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid limit")
				return
			}
			limit = n
		}
		offset := 0
		if raw := r.URL.Query().Get("offset"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				writeError(w, http.StatusBadRequest, "invalid offset")
				return
			}
			offset = n
		}

		approvals, err := app.ListApprovals(r.Context(), statuses, limit, offset)
		if err != nil {
			if errors.Is(err, application.ErrInvalidApprovalStatus) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		if approvals == nil {
			approvals = []approval.Approval{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(approvals)
	}
}
