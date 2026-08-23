package application

import "github.com/google/uuid"

// Finance ApplicationRequests (ROADMAP.md Phase 4 Slice 3). Like
// Research's and M&E's requests, each carries an explicit PrincipalID —
// the real authenticated, resolved Principal the HTTP handler read from
// context, not a fixtures.Registry stand-in.

type CreateBudgetRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	LimitAmount float64
}

type CreateCostConstraintRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
}

type RecordResourceUsageRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	ResultID    uuid.UUID
}

type RunResourceEvaluationRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
}
