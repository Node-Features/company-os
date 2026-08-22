package application

import (
	"context"
	"errors"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
)

// LatestResultView is the status read endpoint's latestResult projection.
type LatestResultView struct {
	ResultID   string         `json:"resultId"`
	Outcome    string         `json:"outcome"`
	ReportedAt string         `json:"reportedAt"`
	Output     map[string]any `json:"output,omitempty"`
}

// AttemptView is one ExecutionAttempt in an ExecutionUnitView's Attempts.
type AttemptView struct {
	AttemptID       string  `json:"attemptId"`
	AttemptNumber   int     `json:"attemptNumber"`
	Status          string  `json:"status"`
	ProviderRunID   *string `json:"providerRunId,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	LastHeartbeatAt *string `json:"lastHeartbeatAt,omitempty"`
	TerminalAt      *string `json:"terminalAt,omitempty"`
}

// ExecutionUnitView is an ExecutionIntent plus every ExecutionAttempt made
// against it, projected for the status read endpoint's live execution
// visualization. Not a "Node" in the runtime/compute-node sense
// (docs/architecture/node.md) — this is per-Workflow dispatch bookkeeping,
// grouped for display.
type ExecutionUnitView struct {
	IntentID     string        `json:"intentId"`
	IntentStatus string        `json:"intentStatus"`
	DueAt        string        `json:"dueAt"`
	Attempts     []AttemptView `json:"attempts"`
}

// WorkflowStatus is the read-only status projection served by the
// GET /v1/workflows/{workflowId} endpoint.
type WorkflowStatus struct {
	Workflow     WorkflowView
	LatestResult *LatestResultView
	Units        []ExecutionUnitView
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func timePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeLayout)
	return &s
}

func executionUnitViewsOf(units []execution.ExecutionUnit) []ExecutionUnitView {
	views := make([]ExecutionUnitView, len(units))
	for i, u := range units {
		attempts := make([]AttemptView, len(u.Attempts))
		for j, a := range u.Attempts {
			attempts[j] = AttemptView{
				AttemptID:       a.AttemptID.String(),
				AttemptNumber:   a.AttemptNumber,
				Status:          string(a.Status),
				ProviderRunID:   a.ProviderRunID,
				CreatedAt:       a.CreatedAt.Format(timeLayout),
				LastHeartbeatAt: timePtr(a.LastHeartbeatAt),
				TerminalAt:      timePtr(a.TerminalAt),
			}
		}
		views[i] = ExecutionUnitView{
			IntentID:     u.IntentID.String(),
			IntentStatus: string(u.IntentStatus),
			DueAt:        u.DueAt.Format(timeLayout),
			Attempts:     attempts,
		}
	}
	return views
}

// GetWorkflowStatus is the read-only status use case behind the status
// endpoint — a projection, not a governed decision (application.md's
// "read-only use cases still enforce organization... but do not invent a
// write transaction").
func (a *Application) GetWorkflowStatus(ctx context.Context, workflowID uuid.UUID) (*WorkflowStatus, error) {
	w, err := a.Repo.LoadWorkflow(ctx, a.Fixtures.Organization().OrganizationID, workflowID)
	if err != nil {
		return nil, err
	}

	status := &WorkflowStatus{Workflow: *viewOf(w)}

	res, err := a.Exec.GetLatestResult(ctx, w.OrganizationID, workflowID)
	if err != nil && !errors.Is(err, ports.ErrNotFound) {
		return nil, err
	}
	if err == nil {
		status.LatestResult = &LatestResultView{
			ResultID:   res.ResultID.String(),
			Outcome:    string(res.Outcome),
			ReportedAt: res.ReportedAt.Format(timeLayout),
			Output:     res.Output,
		}
	}

	units, err := a.Exec.ListExecutionUnits(ctx, w.OrganizationID, workflowID)
	if err != nil {
		return nil, err
	}
	status.Units = executionUnitViewsOf(units)

	return status, nil
}
