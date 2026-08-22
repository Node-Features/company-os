// Package examples holds trivial agent.Agent implementations that exist
// only to exercise agent.Runtime end to end (see the tests in the parent
// agent package). None of these are real CompanyOS agents; they carry no
// Capability, Governance mapping, or AgentPrincipal.
package examples

import (
	"context"
	"log"
)

// EchoAgent is the simplest possible agent.Agent: it logs Input and
// returns nil. It has no knowledge of agent.Runtime's concurrency.Bus at
// all, which is the point — status reporting is Runtime's job, not each
// Agent's.
type EchoAgent struct {
	AgentName string
	Input     string
}

// Name implements agent.Agent.
func (e *EchoAgent) Name() string { return e.AgentName }

// Run implements agent.Agent. It checks ctx once before doing anything, so
// an already-cancelled context is respected even for work this trivial.
func (e *EchoAgent) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	log.Printf("echo agent %q: %s", e.AgentName, e.Input)
	return nil
}
