package application

import (
	"context"
	"errors"

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

// WorkflowStatus is the read-only status projection served by the
// GET /v1/workflows/{workflowId} endpoint.
type WorkflowStatus struct {
	Workflow      WorkflowView
	LatestResult  *LatestResultView
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
			ReportedAt: res.ReportedAt.Format("2006-01-02T15:04:05Z07:00"),
			Output:     res.Output,
		}
	}
	return status, nil
}
