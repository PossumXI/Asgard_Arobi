// Package repositories provides data access layer for database operations.
package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
)

// EthicalDecisionRepository handles ethical decision database operations.
type EthicalDecisionRepository struct {
	db *db.PostgresDB
}

// NewEthicalDecisionRepository creates a new ethical decision repository.
func NewEthicalDecisionRepository(pgDB *db.PostgresDB) *EthicalDecisionRepository {
	return &EthicalDecisionRepository{db: pgDB}
}

// Create inserts a new ethical decision record.
func (r *EthicalDecisionRepository) Create(ctx context.Context, decision *db.EthicalDecision) error {
	query := `
		INSERT INTO ethical_decisions (id, hunoid_id, proposed_action, ethical_assessment, 
		                               decision, reasoning, human_override, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	if decision.ID == uuid.Nil {
		decision.ID = uuid.New()
	}
	if decision.CreatedAt.IsZero() {
		decision.CreatedAt = time.Now().UTC()
	}

	_, err := r.db.ExecContext(ctx, query,
		decision.ID,
		decision.HunoidID,
		decision.ProposedAction,
		decision.EthicalAssessment,
		decision.Decision,
		decision.Reasoning,
		decision.HumanOverride,
		decision.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create ethical decision: %w", err)
	}

	return nil
}

// GetByID retrieves an ethical decision by ID.
func (r *EthicalDecisionRepository) GetByID(ctx context.Context, id string) (*db.EthicalDecision, error) {
	decisionID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid decision ID: %w", err)
	}

	query := `
		SELECT id, hunoid_id, proposed_action, ethical_assessment, 
		       decision, reasoning, human_override, created_at
		FROM ethical_decisions
		WHERE id = $1
	`

	decision := &db.EthicalDecision{}
	var reasoning sql.NullString
	var assessment []byte

	err = r.db.QueryRowContext(ctx, query, decisionID).Scan(
		&decision.ID,
		&decision.HunoidID,
		&decision.ProposedAction,
		&assessment,
		&decision.Decision,
		&reasoning,
		&decision.HumanOverride,
		&decision.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ethical decision not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ethical decision: %w", err)
	}

	decision.Reasoning = reasoning
	decision.EthicalAssessment = assessment

	return decision, nil
}

// GetByHunoidID retrieves ethical decisions for a specific hunoid.
func (r *EthicalDecisionRepository) GetByHunoidID(ctx context.Context, hunoidID string, limit int) ([]*db.EthicalDecision, error) {
	parsedHunoidID, err := uuid.Parse(hunoidID)
	if err != nil {
		return nil, fmt.Errorf("invalid hunoid ID: %w", err)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, hunoid_id, proposed_action, ethical_assessment, 
		       decision, reasoning, human_override, created_at
		FROM ethical_decisions
		WHERE hunoid_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, parsedHunoidID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ethical decisions: %w", err)
	}
	defer rows.Close()

	return r.scanDecisions(rows)
}

// GetByMissionID retrieves ethical decisions for hunoids assigned to a specific mission.
func (r *EthicalDecisionRepository) GetByMissionID(ctx context.Context, missionID string) ([]*db.EthicalDecision, error) {
	parsedMissionID, err := uuid.Parse(missionID)
	if err != nil {
		return nil, fmt.Errorf("invalid mission ID: %w", err)
	}

	// Join with hunoids table to find decisions for hunoids assigned to this mission
	query := `
		SELECT ed.id, ed.hunoid_id, ed.proposed_action, ed.ethical_assessment, 
		       ed.decision, ed.reasoning, ed.human_override, ed.created_at
		FROM ethical_decisions ed
		INNER JOIN hunoids h ON ed.hunoid_id = h.id
		WHERE h.current_mission_id = $1
		ORDER BY ed.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, parsedMissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query ethical decisions by mission: %w", err)
	}
	defer rows.Close()

	return r.scanDecisions(rows)
}

// GetByDecisionType retrieves ethical decisions by decision type (approved, rejected, escalated).
func (r *EthicalDecisionRepository) GetByDecisionType(ctx context.Context, decisionType string, limit int) ([]*db.EthicalDecision, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, hunoid_id, proposed_action, ethical_assessment, 
		       decision, reasoning, human_override, created_at
		FROM ethical_decisions
		WHERE decision = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, decisionType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ethical decisions: %w", err)
	}
	defer rows.Close()

	return r.scanDecisions(rows)
}

// GetRecent retrieves the most recent ethical decisions.
func (r *EthicalDecisionRepository) GetRecent(ctx context.Context, limit int) ([]*db.EthicalDecision, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, hunoid_id, proposed_action, ethical_assessment, 
		       decision, reasoning, human_override, created_at
		FROM ethical_decisions
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent ethical decisions: %w", err)
	}
	defer rows.Close()

	return r.scanDecisions(rows)
}

// GetByDateRange retrieves ethical decisions within a date range.
func (r *EthicalDecisionRepository) GetByDateRange(ctx context.Context, start, end time.Time, limit int) ([]*db.EthicalDecision, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	query := `
		SELECT id, hunoid_id, proposed_action, ethical_assessment, 
		       decision, reasoning, human_override, created_at
		FROM ethical_decisions
		WHERE created_at >= $1 AND created_at <= $2
		ORDER BY created_at DESC
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ethical decisions by date range: %w", err)
	}
	defer rows.Close()

	return r.scanDecisions(rows)
}

// scanDecisions scans rows into ethical decision structs.
func (r *EthicalDecisionRepository) scanDecisions(rows *sql.Rows) ([]*db.EthicalDecision, error) {
	var decisions []*db.EthicalDecision

	for rows.Next() {
		decision := &db.EthicalDecision{}
		var reasoning sql.NullString
		var assessment []byte

		err := rows.Scan(
			&decision.ID,
			&decision.HunoidID,
			&decision.ProposedAction,
			&assessment,
			&decision.Decision,
			&reasoning,
			&decision.HumanOverride,
			&decision.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ethical decision: %w", err)
		}

		decision.Reasoning = reasoning
		decision.EthicalAssessment = assessment

		decisions = append(decisions, decision)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ethical decisions: %w", err)
	}

	return decisions, nil
}

// CountByDecisionType returns counts of decisions grouped by type.
func (r *EthicalDecisionRepository) CountByDecisionType(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT decision, COUNT(*) as count
		FROM ethical_decisions
		GROUP BY decision
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count ethical decisions: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, fmt.Errorf("failed to scan decision count: %w", err)
		}
		counts[decision] = count
	}

	return counts, nil
}
