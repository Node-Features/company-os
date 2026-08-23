package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	knowledgedomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

var validKnowledgeSourceTypes = map[string]knowledgedomain.SourceType{
	"RESEARCH_FINDING": knowledgedomain.SourceResearchFinding,
}

// CaptureKnowledgeCandidateHandler handles POST /v1/knowledge/items/capture.
func CaptureKnowledgeCandidateHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		var body struct {
			SourceType string `json:"sourceType"`
			SourceID   string `json:"sourceId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sourceType, ok := validKnowledgeSourceTypes[body.SourceType]
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid sourceType")
			return
		}
		sourceID, err := uuid.Parse(body.SourceID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid sourceId")
			return
		}
		requestID := uuid.New()
		res := app.CaptureKnowledgeCandidate(r.Context(), application.CaptureKnowledgeCandidateRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			SourceType:  sourceType,
			SourceID:    sourceID,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// RequestKnowledgeApprovalHandler handles
// POST /v1/knowledge/items/{knowledgeItemId}/request-approval. The actual
// approve/reject decision reuses the existing
// POST /v1/approvals/{approvalId}/decide route unchanged — its dispatch is
// already generic by CommandType.
func RequestKnowledgeApprovalHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		itemID, err := uuid.Parse(r.PathValue("knowledgeItemId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid knowledgeItemId")
			return
		}
		var body struct {
			Version       int    `json:"version"`
			ContentDigest string `json:"contentDigest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New()
		res := app.RequestKnowledgeApproval(r.Context(), application.RequestKnowledgeApprovalRequest{
			RequestID:       requestID,
			PrincipalID:     principalID,
			KnowledgeItemID: itemID,
			Version:         body.Version,
			ContentDigest:   body.ContentDigest,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// GetKnowledgeItemHandler handles GET /v1/knowledge/items/{knowledgeItemId}.
func GetKnowledgeItemHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		itemID, err := uuid.Parse(r.PathValue("knowledgeItemId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid knowledgeItemId")
			return
		}
		item, err := app.GetKnowledgeItem(r.Context(), itemID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "knowledge item not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	}
}
