package application

import "github.com/google/uuid"

// M&E ApplicationRequests (ROADMAP.md Phase 4 Slice 2). Like Research's
// requests, each carries an explicit PrincipalID — the real authenticated,
// resolved Principal the HTTP handler read from context, not a
// fixtures.Registry stand-in.

type RecordMetricRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	ResultID    uuid.UUID
}

type RunEvaluationRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	MetricIDs   []uuid.UUID
}
