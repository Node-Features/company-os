package application

import (
	objdomain "github.com/Node-Features/company-os/apps/companyd/internal/domain/objective"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/principal"
	"github.com/google/uuid"
)

// ProposeObjectiveRequest is ROADMAP.md Phase 4 Slice 4's request. Like
// Research's/M&E's/Finance's requests, it carries an explicit
// PrincipalID — the real authenticated, resolved Principal the HTTP
// handler read from context, not a fixtures.Registry stand-in.
// PrincipalKind is that same resolved Principal's Kind, added
// 2026-08-24 (docs/adr/ADR-0010-authority-model-formalization.md) so
// governance.Evaluate can classify requests by real requester kind rather
// than none at all.
type ProposeObjectiveRequest struct {
	RequestID     uuid.UUID
	PrincipalID   uuid.UUID
	PrincipalKind principal.Kind
	SourceType    objdomain.SourceType
	SourceID      uuid.UUID
	Title         string
	Intent        string
}
