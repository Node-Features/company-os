package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// RecordMetricHandler handles POST /v1/me/results/{resultId}/metrics.
func RecordMetricHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		resultID, err := uuid.Parse(r.PathValue("resultId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid resultId")
			return
		}
		requestID := uuid.New()
		res := app.RecordMetric(r.Context(), application.RecordMetricRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			ResultID:    resultID,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// RunEvaluationHandler handles POST /v1/me/evaluations.
func RunEvaluationHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		var body struct {
			MetricIDs []string `json:"metricIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		metricIDs := make([]uuid.UUID, 0, len(body.MetricIDs))
		for _, s := range body.MetricIDs {
			id, err := uuid.Parse(s)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid metricIds")
				return
			}
			metricIDs = append(metricIDs, id)
		}
		requestID := uuid.New()
		res := app.RunEvaluation(r.Context(), application.RunEvaluationRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			MetricIDs:   metricIDs,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// GetPerformanceProfileHandler handles GET /v1/me/subjects/{subjectId}/performance-profile.
func GetPerformanceProfileHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subjectID, err := uuid.Parse(r.PathValue("subjectId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid subjectId")
			return
		}
		profile, err := app.GetPerformanceProfile(r.Context(), subjectID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "performance profile not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(profile)
	}
}
