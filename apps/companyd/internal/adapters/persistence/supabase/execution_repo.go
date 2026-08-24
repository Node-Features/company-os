package supabase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Node-Features/company-os/apps/companyd/internal/domain/execution"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/result"
	"github.com/Node-Features/company-os/apps/companyd/internal/domain/workflow"
	"github.com/Node-Features/company-os/apps/companyd/internal/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ExecutionRepository implements ports.ExecutionRepository: claiming due
// ExecutionIntents under FOR UPDATE SKIP LOCKED, recording attempt status,
// and persisting Results. See docs/domain/execution.md.
type ExecutionRepository struct{ p *Pool }

func NewExecutionRepository(p *Pool) *ExecutionRepository { return &ExecutionRepository{p: p} }

func (r *ExecutionRepository) ClaimDueIntents(ctx context.Context, orgID uuid.UUID, limit int, leaseDuration time.Duration, workerID string) ([]execution.ClaimedExecution, error) {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH claimed AS (
			SELECT intent_id FROM execution_intents
			WHERE status = 'PENDING' AND due_at <= now() AND organization_id = $1
			ORDER BY due_at LIMIT $2 FOR UPDATE SKIP LOCKED
		)
		UPDATE execution_intents SET status = 'CLAIMED'
		WHERE intent_id IN (SELECT intent_id FROM claimed)
		RETURNING intent_id, workflow_id, workflow_version, capability_definition_id,
		          capability_definition_version, governance_decision_id, idempotency_key, constraints, due_at, inputs`,
		orgID, limit)
	if err != nil {
		return nil, err
	}

	type claim struct {
		intent      workflow.ExecutionIntent
		constraints []byte
		inputs      []byte
	}
	var claims []claim
	for rows.Next() {
		var c claim
		c.intent.OrganizationID = orgID
		if err := rows.Scan(&c.intent.IntentID, &c.intent.WorkflowID, &c.intent.WorkflowVersion,
			&c.intent.CapabilityDefinitionID, &c.intent.CapabilityDefinitionVersion, &c.intent.GovernanceDecisionID,
			&c.intent.IdempotencyKey, &c.constraints, &c.intent.DueAt, &c.inputs); err != nil {
			rows.Close()
			return nil, err
		}
		claims = append(claims, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []execution.ClaimedExecution
	for _, c := range claims {
		if len(c.constraints) > 0 {
			_ = json.Unmarshal(c.constraints, &c.intent.Constraints)
		}
		if len(c.inputs) > 0 {
			_ = json.Unmarshal(c.inputs, &c.intent.Inputs)
		}

		var nextAttempt int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(attempt_number),0)+1 FROM execution_attempts WHERE intent_id=$1`, c.intent.IntentID).Scan(&nextAttempt); err != nil {
			return nil, err
		}

		attempt := execution.ExecutionAttempt{
			AttemptID:           uuid.New(),
			OrganizationID:      orgID,
			IntentID:            c.intent.IntentID,
			WorkflowID:          c.intent.WorkflowID,
			WorkflowVersion:     c.intent.WorkflowVersion,
			LogicalOperationID:  c.intent.IntentID.String(),
			AttemptNumber:       nextAttempt,
			CapabilityRequestID: uuid.New(),
			Status:              execution.StatusClaimed,
			CreatedAt:           time.Now().UTC(),
		}
		leaseExpires := attempt.CreatedAt.Add(leaseDuration)
		attempt.LeaseOwner = &workerID
		attempt.LeaseExpiresAt = &leaseExpires

		var fencingToken int64
		if err := tx.QueryRow(ctx, `SELECT nextval('lease_fencing_seq')`).Scan(&fencingToken); err != nil {
			return nil, err
		}
		attempt.LeaseFencingToken = &fencingToken

		_, err = tx.Exec(ctx, `
			INSERT INTO execution_attempts (attempt_id, organization_id, intent_id, workflow_id, workflow_version,
			                                 logical_operation_id, attempt_number, capability_request_id, status,
			                                 lease_owner, lease_fencing_token, lease_expires_at, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
			attempt.AttemptID, attempt.OrganizationID, attempt.IntentID, attempt.WorkflowID, attempt.WorkflowVersion,
			attempt.LogicalOperationID, attempt.AttemptNumber, attempt.CapabilityRequestID, attempt.Status,
			attempt.LeaseOwner, attempt.LeaseFencingToken, attempt.LeaseExpiresAt, attempt.CreatedAt)
		if err != nil {
			return nil, err
		}

		out = append(out, execution.ClaimedExecution{Attempt: attempt, Intent: c.intent})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ExecutionRepository) RecordDispatched(ctx context.Context, attemptID uuid.UUID, fencingToken int64, providerRunID string) error {
	tag, err := r.p.pool.Exec(ctx, `
		UPDATE execution_attempts SET status='DISPATCHED', provider_run_id=$1, last_heartbeat_at=now()
		WHERE attempt_id=$2 AND lease_fencing_token=$3`, providerRunID, attemptID, fencingToken)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		// The fencing token no longer matches this row — either a
		// concurrent writer already moved it (shouldn't happen; only one
		// worker is ever handed this token), or ReclaimExpiredLeases
		// already reclaimed this attempt (lease expired, token bumped) and
		// this caller is the original, since-superseded worker calling in
		// late. Either way this must fail closed, not silently succeed or
		// silently no-op, so the caller (Runtime) aborts rather than
		// proceeding as if it still held the lease. ports.ErrConflict
		// matches every other compare-and-swap loss in this codebase
		// (CommitTransition, ResolveApproval, KnowledgeRepository.
		// TransitionStatus) rather than an ad hoc error type.
		return fmt.Errorf("execution_repo: lease lost, attempt not dispatched: %w", ports.ErrConflict)
	}
	return nil
}

func (r *ExecutionRepository) RecordTerminal(ctx context.Context, attemptID uuid.UUID, fencingToken int64, status execution.AttemptStatus, resultID *uuid.UUID) error {
	tag, err := r.p.pool.Exec(ctx, `
		UPDATE execution_attempts SET status=$1, terminal_at=now(), result_id=$2
		WHERE attempt_id=$3 AND lease_fencing_token=$4`, status, resultID, attemptID, fencingToken)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		// Same reasoning as RecordDispatched above — most commonly this is
		// the original worker's terminal report arriving after
		// ReclaimExpiredLeases already reclaimed (and thus fenced out) this
		// attempt. Must fail closed: the caller must not treat this as "my
		// result is now durable" when it isn't.
		return fmt.Errorf("execution_repo: lease lost, attempt not recorded: %w", ports.ErrConflict)
	}
	return nil
}

func (r *ExecutionRepository) ScheduleRetry(ctx context.Context, orgID, intentID uuid.UUID, dueAt time.Time) error {
	_, err := r.p.pool.Exec(ctx, `UPDATE execution_intents SET status='PENDING', due_at=$1 WHERE organization_id=$2 AND intent_id=$3`, dueAt, orgID, intentID)
	return err
}

// ReclaimExpiredLeases implements ports.ExecutionRepository.ReclaimExpiredLeases
// — see that doc comment for the fencing-token safety argument. The WHERE
// clause's status set (CLAIMED, DISPATCHED, WAITING) matches exactly the
// three FROM-states execution.AttemptStatus.CanTransitionTo's table
// legalizes a transition to LEASE_EXPIRED from; reclaim never touches a row
// already in a genuinely terminal status (SUCCEEDED, FAILED_TERMINAL,
// CANCELLED, or already LEASE_EXPIRED), so a real completion that lands
// concurrently with a reclaim sweep simply excludes that row from this
// query (its status no longer matches) rather than racing destructively.
func (r *ExecutionRepository) ReclaimExpiredLeases(ctx context.Context, orgID uuid.UUID, limit int) ([]execution.ClaimedExecution, error) {
	tx, err := r.p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH expired AS (
			SELECT attempt_id FROM execution_attempts
			WHERE organization_id = $1 AND status IN ('CLAIMED', 'DISPATCHED', 'WAITING') AND lease_expires_at < now()
			ORDER BY lease_expires_at LIMIT $2 FOR UPDATE SKIP LOCKED
		)
		UPDATE execution_attempts SET status = 'LEASE_EXPIRED', terminal_at = now(),
		       lease_fencing_token = nextval('lease_fencing_seq')
		WHERE attempt_id IN (SELECT attempt_id FROM expired)
		RETURNING attempt_id, organization_id, intent_id, workflow_id, workflow_version, logical_operation_id,
		          attempt_number, capability_request_id, lease_owner, provider_run_id, created_at, last_heartbeat_at`,
		orgID, limit)
	if err != nil {
		return nil, err
	}

	var attempts []execution.ExecutionAttempt
	intentIDs := map[uuid.UUID]bool{}
	for rows.Next() {
		var a execution.ExecutionAttempt
		if err := rows.Scan(&a.AttemptID, &a.OrganizationID, &a.IntentID, &a.WorkflowID, &a.WorkflowVersion,
			&a.LogicalOperationID, &a.AttemptNumber, &a.CapabilityRequestID, &a.LeaseOwner, &a.ProviderRunID,
			&a.CreatedAt, &a.LastHeartbeatAt); err != nil {
			rows.Close()
			return nil, err
		}
		a.Status = execution.StatusLeaseExpired
		attempts = append(attempts, a)
		intentIDs[a.IntentID] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(attempts) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	}

	ids := make([]string, 0, len(intentIDs))
	for id := range intentIDs {
		ids = append(ids, id.String())
	}
	intentRows, err := tx.Query(ctx, `
		SELECT intent_id, workflow_id, workflow_version, capability_definition_id, capability_definition_version,
		       governance_decision_id, idempotency_key, constraints, due_at, inputs
		FROM execution_intents WHERE intent_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	intentsByID := map[uuid.UUID]workflow.ExecutionIntent{}
	for intentRows.Next() {
		var intent workflow.ExecutionIntent
		var constraints, inputs []byte
		if err := intentRows.Scan(&intent.IntentID, &intent.WorkflowID, &intent.WorkflowVersion,
			&intent.CapabilityDefinitionID, &intent.CapabilityDefinitionVersion, &intent.GovernanceDecisionID,
			&intent.IdempotencyKey, &constraints, &intent.DueAt, &inputs); err != nil {
			intentRows.Close()
			return nil, err
		}
		intent.OrganizationID = orgID
		if len(constraints) > 0 {
			_ = json.Unmarshal(constraints, &intent.Constraints)
		}
		if len(inputs) > 0 {
			_ = json.Unmarshal(inputs, &intent.Inputs)
		}
		intentsByID[intent.IntentID] = intent
	}
	intentRows.Close()
	if err := intentRows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	out := make([]execution.ClaimedExecution, 0, len(attempts))
	for _, a := range attempts {
		out = append(out, execution.ClaimedExecution{Attempt: a, Intent: intentsByID[a.IntentID]})
	}
	return out, nil
}

func (r *ExecutionRepository) SaveResult(ctx context.Context, res *result.Result) error {
	output, _ := json.Marshal(res.Output)
	_, err := r.p.pool.Exec(ctx, `
		INSERT INTO results (result_id, organization_id, result_type, workflow_id, objective_id, intent_id,
		                      attempt_id, capability_request_id, idempotency_key, producing_principal_id,
		                      provider_adapter, model_id, outcome, output, error_classification, retryable,
		                      started_at, observed_at, reported_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`,
		res.ResultID, res.OrganizationID, res.ResultType, res.WorkflowID, res.ObjectiveID, res.IntentID,
		res.AttemptID, res.CapabilityRequestID, res.IdempotencyKey, res.ProducingPrincipalID, res.ProviderAdapter,
		res.ModelID, res.Outcome, output, res.ErrorClassification, res.Retryable, res.StartedAt, res.ObservedAt, res.ReportedAt)
	return err
}

func (r *ExecutionRepository) GetResult(ctx context.Context, orgID, resultID uuid.UUID) (*result.Result, error) {
	row := r.p.pool.QueryRow(ctx, `
		SELECT result_id, organization_id, result_type, workflow_id, objective_id, intent_id, attempt_id,
		       capability_request_id, idempotency_key, producing_principal_id, provider_adapter, model_id,
		       outcome, output, error_classification, retryable, started_at, observed_at, reported_at, accepted, decided_at
		FROM results WHERE organization_id=$1 AND result_id=$2`, orgID, resultID)

	var res result.Result
	var output []byte
	if err := row.Scan(&res.ResultID, &res.OrganizationID, &res.ResultType, &res.WorkflowID, &res.ObjectiveID,
		&res.IntentID, &res.AttemptID, &res.CapabilityRequestID, &res.IdempotencyKey, &res.ProducingPrincipalID,
		&res.ProviderAdapter, &res.ModelID, &res.Outcome, &output, &res.ErrorClassification, &res.Retryable,
		&res.StartedAt, &res.ObservedAt, &res.ReportedAt, &res.Accepted, &res.DecidedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if len(output) > 0 {
		_ = json.Unmarshal(output, &res.Output)
	}
	return &res, nil
}

// ListExecutionUnits serves ports.ExecutionRepository's status-projection
// read: every ExecutionIntent for workflowID plus every ExecutionAttempt
// made against each, oldest first. Two queries rather than a join — intents
// without any attempt yet (freshly claimed, or still PENDING) must still
// appear as a unit with an empty Attempts slice.
func (r *ExecutionRepository) ListExecutionUnits(ctx context.Context, orgID, workflowID uuid.UUID) ([]execution.ExecutionUnit, error) {
	rows, err := r.p.pool.Query(ctx, `
		SELECT intent_id, status, due_at FROM execution_intents
		WHERE organization_id=$1 AND workflow_id=$2 ORDER BY created_at`, orgID, workflowID)
	if err != nil {
		return nil, err
	}
	unitsByIntent := map[uuid.UUID]*execution.ExecutionUnit{}
	var order []uuid.UUID
	for rows.Next() {
		var u execution.ExecutionUnit
		if err := rows.Scan(&u.IntentID, &u.IntentStatus, &u.DueAt); err != nil {
			rows.Close()
			return nil, err
		}
		unitsByIntent[u.IntentID] = &u
		order = append(order, u.IntentID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(order) == 0 {
		return nil, nil
	}

	attemptRows, err := r.p.pool.Query(ctx, `
		SELECT attempt_id, organization_id, intent_id, workflow_id, workflow_version, logical_operation_id,
		       attempt_number, capability_request_id, status, lease_owner, lease_fencing_token, lease_expires_at,
		       provider_run_id, created_at, last_heartbeat_at, next_attempt_due_at, terminal_at, result_id
		FROM execution_attempts WHERE organization_id=$1 AND workflow_id=$2 ORDER BY created_at`, orgID, workflowID)
	if err != nil {
		return nil, err
	}
	defer attemptRows.Close()
	for attemptRows.Next() {
		var a execution.ExecutionAttempt
		if err := attemptRows.Scan(&a.AttemptID, &a.OrganizationID, &a.IntentID, &a.WorkflowID, &a.WorkflowVersion,
			&a.LogicalOperationID, &a.AttemptNumber, &a.CapabilityRequestID, &a.Status, &a.LeaseOwner,
			&a.LeaseFencingToken, &a.LeaseExpiresAt, &a.ProviderRunID, &a.CreatedAt, &a.LastHeartbeatAt,
			&a.NextAttemptDueAt, &a.TerminalAt, &a.ResultID); err != nil {
			return nil, err
		}
		if u, ok := unitsByIntent[a.IntentID]; ok {
			u.Attempts = append(u.Attempts, a)
		}
	}
	if err := attemptRows.Err(); err != nil {
		return nil, err
	}

	out := make([]execution.ExecutionUnit, 0, len(order))
	for _, id := range order {
		out = append(out, *unitsByIntent[id])
	}
	return out, nil
}

func (r *ExecutionRepository) GetLatestResult(ctx context.Context, orgID, workflowID uuid.UUID) (*result.Result, error) {
	row := r.p.pool.QueryRow(ctx, `
		SELECT result_id, organization_id, result_type, workflow_id, objective_id, intent_id, attempt_id,
		       capability_request_id, idempotency_key, producing_principal_id, provider_adapter, model_id,
		       outcome, output, error_classification, retryable, started_at, observed_at, reported_at, accepted, decided_at
		FROM results WHERE organization_id=$1 AND workflow_id=$2 ORDER BY reported_at DESC LIMIT 1`, orgID, workflowID)

	var res result.Result
	var output []byte
	if err := row.Scan(&res.ResultID, &res.OrganizationID, &res.ResultType, &res.WorkflowID, &res.ObjectiveID,
		&res.IntentID, &res.AttemptID, &res.CapabilityRequestID, &res.IdempotencyKey, &res.ProducingPrincipalID,
		&res.ProviderAdapter, &res.ModelID, &res.Outcome, &output, &res.ErrorClassification, &res.Retryable,
		&res.StartedAt, &res.ObservedAt, &res.ReportedAt, &res.Accepted, &res.DecidedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if len(output) > 0 {
		_ = json.Unmarshal(output, &res.Output)
	}
	return &res, nil
}
