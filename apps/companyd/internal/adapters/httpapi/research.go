package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// departmentCommandResponse is the shared response shape for every
// state-changing Research (docs/workflows/research-loop.md) and M&E
// (ROADMAP.md Phase 4 Slice 2) endpoint, mirroring workflowCommandResponse's
// role for Workflow endpoints.
type departmentCommandResponse struct {
	Outcome    string   `json:"outcome"`
	ResourceID string   `json:"resourceId,omitempty"`
	// ApprovalID is set only when Outcome is APPROVAL_REQUIRED — first
	// exercised by ROADMAP.md Phase 4 Slice 4's ProposeObjective, whose
	// policy always requires human approval; every prior department
	// endpoint is AUTOMATIC-only and never sets it.
	ApprovalID string   `json:"approvalId,omitempty"`
	RequestID  string   `json:"requestId"`
	Reasons    []string `json:"reasons,omitempty"`
}

func writeDepartmentResult(w http.ResponseWriter, requestID uuid.UUID, res application.Result) {
	resp := departmentCommandResponse{
		Outcome:   string(res.Outcome),
		RequestID: requestID.String(),
		Reasons:   res.Reasons,
	}
	if res.ResourceID != nil {
		resp.ResourceID = res.ResourceID.String()
	}
	if res.ApprovalID != nil {
		resp.ApprovalID = res.ApprovalID.String()
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCodeFor(res.Outcome))
	json.NewEncoder(w).Encode(resp)
}

// requirePrincipal reads the Principal ResolvePrincipal resolved for this
// request — Research and M&E actions use the real signed-in Principal, not
// a fixtures.Registry stand-in. Absence is a programmer error — every
// route using this is wired behind RequireHumanAuth+ResolvePrincipal — not
// a caller fault.
func requirePrincipal(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	p, ok := PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return uuid.UUID{}, false
	}
	return p.PrincipalID, true
}

// SubmitSignalHandler handles POST /v1/research/signals.
func SubmitSignalHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		var body struct {
			SourceType  string `json:"sourceType"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New()
		res := app.SubmitSignal(r.Context(), application.SubmitSignalRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			SourceType:  body.SourceType,
			Description: body.Description,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// OpenResearchQuestionHandler handles POST /v1/research/signals/{signalId}/questions.
func OpenResearchQuestionHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		signalID, err := uuid.Parse(r.PathValue("signalId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid signalId")
			return
		}
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New()
		res := app.OpenResearchQuestion(r.Context(), application.OpenResearchQuestionRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			SignalID:    signalID,
			Text:        body.Text,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// RecordEvidenceHandler handles POST /v1/research/questions/{questionId}/evidence.
func RecordEvidenceHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		questionID, err := uuid.Parse(r.PathValue("questionId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid questionId")
			return
		}
		var body struct {
			Source      string     `json:"source"`
			Content     string     `json:"content"`
			RetrievedAt *time.Time `json:"retrievedAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		var retrievedAt time.Time
		if body.RetrievedAt != nil {
			retrievedAt = *body.RetrievedAt
		}
		requestID := uuid.New()
		res := app.RecordEvidence(r.Context(), application.RecordEvidenceRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			QuestionID:  questionID,
			Source:      body.Source,
			Content:     body.Content,
			RetrievedAt: retrievedAt,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// PublishFindingHandler handles POST /v1/research/questions/{questionId}/findings.
func PublishFindingHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		questionID, err := uuid.Parse(r.PathValue("questionId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid questionId")
			return
		}
		var body struct {
			Claim       string   `json:"claim"`
			EvidenceIDs []string `json:"evidenceIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		evidenceIDs := make([]uuid.UUID, 0, len(body.EvidenceIDs))
		for _, s := range body.EvidenceIDs {
			id, err := uuid.Parse(s)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid evidenceIds")
				return
			}
			evidenceIDs = append(evidenceIDs, id)
		}
		requestID := uuid.New()
		res := app.PublishFinding(r.Context(), application.PublishFindingRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			QuestionID:  questionID,
			Claim:       body.Claim,
			EvidenceIDs: evidenceIDs,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// IssueRecommendationHandler handles POST /v1/research/findings/{findingId}/recommendations.
func IssueRecommendationHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		findingID, err := uuid.Parse(r.PathValue("findingId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid findingId")
			return
		}
		var body struct {
			ProposedAction string `json:"proposedAction"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New()
		res := app.IssueRecommendation(r.Context(), application.IssueRecommendationRequest{
			RequestID:      requestID,
			PrincipalID:    principalID,
			FindingID:      findingID,
			ProposedAction: body.ProposedAction,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// GetResearchQuestionHandler handles GET /v1/research/questions/{questionId}.
func GetResearchQuestionHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		questionID, err := uuid.Parse(r.PathValue("questionId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid questionId")
			return
		}
		status, err := app.GetResearchQuestionStatus(r.Context(), questionID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "research question not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}
