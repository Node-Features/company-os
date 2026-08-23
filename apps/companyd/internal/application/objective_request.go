package application

import (
	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/google/uuid"
)

// ProposeObjectiveRequest is ROADMAP.md Phase 4 Slice 4's request. Like
// Research's/M&E's/Finance's requests, it carries an explicit
// PrincipalID — the real authenticated, resolved Principal the HTTP
// handler read from context, not a fixtures.Registry stand-in.
type ProposeObjectiveRequest struct {
	RequestID   uuid.UUID
	PrincipalID uuid.UUID
	SourceType  objdomain.SourceType
	SourceID    uuid.UUID
	Title       string
	Intent      string
}
