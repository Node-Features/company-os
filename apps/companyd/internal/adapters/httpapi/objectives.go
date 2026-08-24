package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

var validSourceTypes = map[string]objdomain.SourceType{
	"FINDING":             objdomain.SourceFinding,
	"RECOMMENDATION":      objdomain.SourceRecommendation,
	"EVALUATION":          objdomain.SourceEvaluation,
	"RESOURCE_EVALUATION": objdomain.SourceResourceEvaluation,
}

// ProposeObjectiveHandler handles POST /v1/objectives/propose.
func ProposeObjectiveHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, ok := PrincipalFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		var body struct {
			SourceType string `json:"sourceType"`
			SourceID   string `json:"sourceId"`
			Title      string `json:"title"`
			Intent     string `json:"intent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sourceType, ok := validSourceTypes[body.SourceType]
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
		res := app.ProposeObjective(r.Context(), application.ProposeObjectiveRequest{
			RequestID:     requestID,
			PrincipalID:   p.PrincipalID,
			PrincipalKind: p.Kind,
			SourceType:    sourceType,
			SourceID:      sourceID,
			Title:         body.Title,
			Intent:        body.Intent,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// GetObjectiveHandler handles GET /v1/objectives/{objectiveId}.
func GetObjectiveHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		objectiveID, err := uuid.Parse(r.PathValue("objectiveId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid objectiveId")
			return
		}
		obj, err := app.GetObjective(r.Context(), objectiveID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "objective not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(obj)
	}
}
