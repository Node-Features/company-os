package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// workflowCommandResponse is the shared response shape for every
// state-changing endpoint below, matching apps/web/lib/companyd-client.ts's
// documented outcome vocabulary exactly.
type workflowCommandResponse struct {
	Outcome    string   `json:"outcome"`
	WorkflowID string   `json:"workflowId,omitempty"`
	Version    int64    `json:"version,omitempty"`
	State      string   `json:"state,omitempty"`
	RequestID  string   `json:"requestId"`
	Reasons    []string `json:"reasons,omitempty"`
	// ApprovalID is set only when Outcome is APPROVAL_REQUIRED — the
	// client needs it to later call POST /v1/approvals/{approvalId}/decide.
	ApprovalID string `json:"approvalId,omitempty"`
}

// statusCodeFor is first-slice plan decision #11's HTTP status mapping for
// the 8 normalized outcomes.
func statusCodeFor(outcome application.Outcome) int {
	switch outcome {
	case application.Accepted:
		return http.StatusOK
	case application.ApprovalRequired, application.Indeterminate:
		return http.StatusAccepted
	case application.Rejected, application.Invalid:
		return http.StatusBadRequest
	case application.Denied:
		return http.StatusForbidden
	case application.Conflict:
		return http.StatusConflict
	case application.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeResult(w http.ResponseWriter, requestID uuid.UUID, res application.Result) {
	resp := workflowCommandResponse{
		Outcome:   string(res.Outcome),
		RequestID: requestID.String(),
		Reasons:   res.Reasons,
	}
	if res.Workflow != nil {
		resp.WorkflowID = res.Workflow.WorkflowID
		resp.Version = res.Workflow.Version
		resp.State = res.Workflow.State
	}
	if res.ApprovalID != nil {
		resp.ApprovalID = res.ApprovalID.String()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCodeFor(res.Outcome))
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// CreateWorkflowHandler handles POST /v1/workflows.
func CreateWorkflowHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		var body struct {
			IdempotencyKey string `json:"idempotencyKey"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.IdempotencyKey == "" {
			body.IdempotencyKey = uuid.New().String()
		}
		requestID := uuid.New()
		res := app.CreateWorkflow(r.Context(), application.CreateWorkflowRequest{
			RequestID:             requestID,
			IdempotencyKey:        body.IdempotencyKey,
			RequestingPrincipalID: principalID,
		})
		writeResult(w, requestID, res)
	}
}

// StartWorkflowHandler handles POST /v1/workflows/{workflowId}/start.
func StartWorkflowHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		workflowID, err := uuid.Parse(r.PathValue("workflowId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workflowId")
			return
		}
		var body struct {
			ExpectedVersion int64  `json:"expectedVersion"`
			IdempotencyKey  string `json:"idempotencyKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.IdempotencyKey == "" {
			body.IdempotencyKey = uuid.New().String()
		}
		requestID := uuid.New()
		res := app.StartWorkflow(r.Context(), application.StartWorkflowRequest{
			RequestID:             requestID,
			IdempotencyKey:        body.IdempotencyKey,
			WorkflowID:            workflowID,
			ExpectedVersion:       body.ExpectedVersion,
			RequestingPrincipalID: principalID,
		})
		writeResult(w, requestID, res)
	}
}

// resultDecisionHandler backs both accept and reject — exposed over HTTP in
// addition to Runtime's in-process call purely so a manual curl runbook can
// exercise the pipeline independent of a live provider call (first-slice
// plan decision #15). Runtime's production path calls Application.SubmitResult
// directly, never through this HTTP hop.
func resultDecisionHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ResultID        string `json:"resultId"`
			ExpectedVersion int64  `json:"expectedVersion"`
			IdempotencyKey  string `json:"idempotencyKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		resultID, err := uuid.Parse(body.ResultID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid resultId")
			return
		}
		if body.IdempotencyKey == "" {
			body.IdempotencyKey = uuid.New().String()
		}
		requestID := uuid.New()
		res := app.SubmitResult(r.Context(), application.SubmitResultRequest{
			RequestID:       requestID,
			IdempotencyKey:  body.IdempotencyKey,
			ResultID:        resultID,
			ExpectedVersion: body.ExpectedVersion,
		})
		writeResult(w, requestID, res)
	}
}

// AcceptResultHandler handles POST /v1/workflows/{workflowId}/results/accept.
func AcceptResultHandler(app *application.Application) http.HandlerFunc {
	return resultDecisionHandler(app)
}

// RejectResultHandler handles POST /v1/workflows/{workflowId}/results/reject.
func RejectResultHandler(app *application.Application) http.HandlerFunc {
	return resultDecisionHandler(app)
}

// CancelWorkflowHandler handles POST /v1/workflows/{workflowId}/cancel.
func CancelWorkflowHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		workflowID, err := uuid.Parse(r.PathValue("workflowId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workflowId")
			return
		}
		var body struct {
			ExpectedVersion int64  `json:"expectedVersion"`
			IdempotencyKey  string `json:"idempotencyKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if body.IdempotencyKey == "" {
			body.IdempotencyKey = uuid.New().String()
		}
		requestID := uuid.New()
		res := app.CancelWorkflow(r.Context(), application.CancelWorkflowRequest{
			RequestID:             requestID,
			IdempotencyKey:        body.IdempotencyKey,
			WorkflowID:            workflowID,
			ExpectedVersion:       body.ExpectedVersion,
			RequestingPrincipalID: principalID,
		})
		writeResult(w, requestID, res)
	}
}

// GetWorkflowHandler handles GET /v1/workflows/{workflowId}.
func GetWorkflowHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		workflowID, err := uuid.Parse(r.PathValue("workflowId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workflowId")
			return
		}
		status, err := app.GetWorkflowStatus(r.Context(), workflowID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "workflow not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		resp := struct {
			WorkflowID     string                          `json:"workflowId"`
			State          string                          `json:"state"`
			Version        int64                           `json:"version"`
			WaitReason     *string                         `json:"waitReason"`
			TerminalReason *string                         `json:"terminalReason"`
			LatestResult   *application.LatestResultView   `json:"latestResult,omitempty"`
			Units          []application.ExecutionUnitView `json:"units"`
		}{
			WorkflowID:     status.Workflow.WorkflowID,
			State:          status.Workflow.State,
			Version:        status.Workflow.Version,
			WaitReason:     status.Workflow.WaitReason,
			TerminalReason: status.Workflow.TerminalReason,
			LatestResult:   status.LatestResult,
			Units:          status.Units,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
