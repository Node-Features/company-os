package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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

// ListObjectivesHandler handles GET /v1/objectives. Query params: statuses
// (comma-separated, e.g. "PROPOSED,RETIRED"; omitted means every status),
// limit, offset (real page-based pagination). See application.ListObjectives's
// doc comment: this can only ever return already-approved proposals (a
// pending or rejected proposal has no Objective row) — a UI listing
// "Decisions" still needs the Approvals list for pending/rejected ones.
func ListObjectivesHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var statuses []objdomain.Status
		if raw := r.URL.Query().Get("statuses"); raw != "" {
			for _, s := range strings.Split(raw, ",") {
				statuses = append(statuses, objdomain.Status(strings.TrimSpace(s)))
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

		objectives, err := app.ListObjectives(r.Context(), statuses, limit, offset)
		if err != nil {
			if errors.Is(err, application.ErrInvalidObjectiveStatus) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		if objectives == nil {
			objectives = []objdomain.Objective{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(objectives)
	}
}
