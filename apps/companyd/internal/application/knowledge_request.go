package application

import (
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/knowledge"
	"github.com/google/uuid"
)

// CaptureKnowledgeCandidateRequest is the CAPTURE_KNOWLEDGE_CANDIDATE use
// case's input (docs/architecture/knowledge.md's ingestion steps 1-4). Only
// a source reference is accepted — never client-supplied claim content —
// so captured content always comes from the authoritative source.
type CaptureKnowledgeCandidateRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	SourceType  knowledge.SourceType
	SourceID    uuid.UUID
}
