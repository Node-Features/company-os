package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Node-Features/company-os/apps/companyd/internal/application"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// CreateBudgetHandler handles POST /v1/finance/budgets.
func CreateBudgetHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		var body struct {
			LimitAmount float64 `json:"limitAmount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		requestID := uuid.New()
		res := app.CreateBudget(r.Context(), application.CreateBudgetRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			LimitAmount: body.LimitAmount,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// CreateCostConstraintHandler handles POST /v1/finance/constraints.
func CreateCostConstraintHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		requestID := uuid.New()
		res := app.CreateCostConstraint(r.Context(), application.CreateCostConstraintRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// RecordResourceUsageHandler handles POST /v1/finance/results/{resultId}/usage.
func RecordResourceUsageHandler(app *application.Application) http.HandlerFunc {
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
		res := app.RecordResourceUsage(r.Context(), application.RecordResourceUsageRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
			ResultID:    resultID,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// RunResourceEvaluationHandler handles POST /v1/finance/evaluations.
func RunResourceEvaluationHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principalID, ok := requirePrincipal(w, r)
		if !ok {
			return
		}
		requestID := uuid.New()
		res := app.RunResourceEvaluation(r.Context(), application.RunResourceEvaluationRequest{
			RequestID:   requestID,
			PrincipalID: principalID,
		})
		writeDepartmentResult(w, requestID, res)
	}
}

// GetConstraintStatusHandler handles GET /v1/finance/constraint-status.
func GetConstraintStatusHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := app.GetConstraintStatus(r.Context())
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// GetResourceEvaluationHandler handles GET /v1/finance/evaluations/{evaluationId}.
func GetResourceEvaluationHandler(app *application.Application) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		evaluationID, err := uuid.Parse(r.PathValue("evaluationId"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid evaluationId")
			return
		}
		evaluation, err := app.GetResourceEvaluation(r.Context(), evaluationID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				writeError(w, http.StatusNotFound, "resource evaluation not found")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(evaluation)
	}
}
