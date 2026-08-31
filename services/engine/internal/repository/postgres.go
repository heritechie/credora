package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"credora/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresRepository implements AssessmentRepository using PostgreSQL.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL-backed assessment repository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// ConnectPostgres opens a connection to PostgreSQL using the given DSN.
func ConnectPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Run migrations.
	if err := migrate(db, "postgres"); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func (r *PostgresRepository) Create(ctx context.Context, a domain.Assessment) error {
	scoreJSON, err := jsonScore(a.Score)
	if err != nil {
		return fmt.Errorf("marshal score: %w", err)
	}

	outputsJSON, err := jsonOutputs(a.Decision)
	if err != nil {
		return fmt.Errorf("marshal outputs: %w", err)
	}

	var appID, appPurpose sql.NullString
	var appAmount sql.NullInt64
	if a.Application != nil {
		appID = sql.NullString{String: a.Application.ID, Valid: a.Application.ID != ""}
		appPurpose = sql.NullString{String: a.Application.Purpose, Valid: a.Application.Purpose != ""}
		if a.Application.RequestedAmount != nil {
			appAmount = sql.NullInt64{Int64: *a.Application.RequestedAmount, Valid: true}
		}
	}

	var decisionOutcome sql.NullString
	var decisionPolicyID string
	var decisionPolicyVersion int
	if a.Decision != nil {
		decisionOutcome = sql.NullString{String: a.Decision.Outcome.String(), Valid: true}
		decisionPolicyID = a.Decision.PolicyID
		decisionPolicyVersion = a.Decision.PolicyVersion
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO assessments (
			id, applicant_id, applicant_name, applicant_age,
			application_id, application_requested_amount, application_purpose,
			score, status, policy_id, policy_version,
			decision_outcome, decision_outputs, decision_policy_id, decision_policy_version,
			error, created_at, started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		a.ID, a.Applicant.ID, a.Applicant.Name, a.Applicant.Age,
		appID, appAmount, appPurpose,
		scoreJSON, a.Status.String(), a.PolicyID, a.PolicyVersion,
		decisionOutcome, outputsJSON, decisionPolicyID, decisionPolicyVersion,
		nullString(a.Error), a.CreatedAt, a.StartedAt, a.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("insert assessment: %w", err)
	}

	if a.Decision != nil {
		if err := r.insertDecision(ctx, a.ID, *a.Decision); err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
	}

	if err := r.insertEvidence(ctx, a.ID, a.Evidence); err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id string) (domain.Assessment, error) {
	var a domain.Assessment
	var status string
	var scoreJSON sql.NullString
	var outputsJSON sql.NullString
	var errorStr sql.NullString
	var startedAt, completedAt sql.NullTime
	var appID, appPurpose sql.NullString
	var appAmount sql.NullInt64
	var decisionOutcome sql.NullString
	var decisionPolicyID string
	var decisionPolicyVersion int

	err := r.db.QueryRowContext(ctx, `
		SELECT id, applicant_id, applicant_name, applicant_age,
			application_id, application_requested_amount, application_purpose,
			score, status, policy_id, policy_version,
			decision_outcome, decision_outputs, decision_policy_id, decision_policy_version,
			error, created_at, started_at, completed_at
		FROM assessments WHERE id = $1`, id,
	).Scan(
		&a.ID, &a.Applicant.ID, &a.Applicant.Name, &a.Applicant.Age,
		&appID, &appAmount, &appPurpose,
		&scoreJSON, &status, &a.PolicyID, &a.PolicyVersion,
		&decisionOutcome, &outputsJSON, &decisionPolicyID, &decisionPolicyVersion,
		&errorStr, &a.CreatedAt, &startedAt, &completedAt,
	)
	if err == sql.ErrNoRows {
		return domain.Assessment{}, fmt.Errorf("assessment %s not found", id)
	}
	if err != nil {
		return domain.Assessment{}, fmt.Errorf("get assessment: %w", err)
	}

	// Reconstruct optional Application
	if appID.Valid && appID.String != "" {
		app := &domain.Application{
			ID:      appID.String,
			Purpose: appPurpose.String,
		}
		if appAmount.Valid {
			app.RequestedAmount = &appAmount.Int64
		}
		a.Application = app
	}

	a.Status = parseStatus(status)
	if errorStr.Valid {
		a.Error = errorStr.String
	}
	if startedAt.Valid {
		a.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		a.CompletedAt = &completedAt.Time
	}
	if scoreJSON.Valid && scoreJSON.String != "" {
		var score domain.CreditScore
		if err := json.Unmarshal([]byte(scoreJSON.String), &score); err != nil {
			return domain.Assessment{}, fmt.Errorf("unmarshal score: %w", err)
		}
		a.Score = &score
	}

	// Reconstruct Decision
	if decisionOutcome.Valid && decisionOutcome.String != "" {
		d := &domain.Decision{
			Outcome:       parseDecisionOutcome(decisionOutcome.String),
			PolicyID:      decisionPolicyID,
			PolicyVersion: decisionPolicyVersion,
		}

		if outputsJSON.Valid && outputsJSON.String != "" {
			var outputs domain.DecisionOutputs
			if err := json.Unmarshal([]byte(outputsJSON.String), &outputs); err != nil {
				return domain.Assessment{}, fmt.Errorf("unmarshal outputs: %w", err)
			}
			d.Outputs = &outputs
		}

		reasons, err := r.getDecision(ctx, id)
		if err != nil {
			return domain.Assessment{}, fmt.Errorf("get decision: %w", err)
		}
		d.Reasons = reasons

		a.Decision = d
	}

	a.Evidence, err = r.getEvidence(ctx, id)
	if err != nil {
		return domain.Assessment{}, fmt.Errorf("get evidence: %w", err)
	}

	return a, nil
}

func (r *PostgresRepository) Update(ctx context.Context, a domain.Assessment) error {
	scoreJSON, err := jsonScore(a.Score)
	if err != nil {
		return fmt.Errorf("marshal score: %w", err)
	}

	outputsJSON, err := jsonOutputs(a.Decision)
	if err != nil {
		return fmt.Errorf("marshal outputs: %w", err)
	}

	var appID, appPurpose sql.NullString
	var appAmount sql.NullInt64
	if a.Application != nil {
		appID = sql.NullString{String: a.Application.ID, Valid: a.Application.ID != ""}
		appPurpose = sql.NullString{String: a.Application.Purpose, Valid: a.Application.Purpose != ""}
		if a.Application.RequestedAmount != nil {
			appAmount = sql.NullInt64{Int64: *a.Application.RequestedAmount, Valid: true}
		}
	}

	var decisionOutcome sql.NullString
	var decisionPolicyID string
	var decisionPolicyVersion int
	if a.Decision != nil {
		decisionOutcome = sql.NullString{String: a.Decision.Outcome.String(), Valid: true}
		decisionPolicyID = a.Decision.PolicyID
		decisionPolicyVersion = a.Decision.PolicyVersion
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE assessments SET
			applicant_id = $2, applicant_name = $3, applicant_age = $4,
			application_id = $5, application_requested_amount = $6, application_purpose = $7,
			score = $8, status = $9, policy_id = $10, policy_version = $11,
			decision_outcome = $12, decision_outputs = $13, decision_policy_id = $14, decision_policy_version = $15,
			error = $16, started_at = $17, completed_at = $18
		WHERE id = $1`,
		a.ID, a.Applicant.ID, a.Applicant.Name, a.Applicant.Age,
		appID, appAmount, appPurpose,
		scoreJSON, a.Status.String(), a.PolicyID, a.PolicyVersion,
		decisionOutcome, outputsJSON, decisionPolicyID, decisionPolicyVersion,
		nullString(a.Error), a.StartedAt, a.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("update assessment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("assessment %s not found", a.ID)
	}

	// Replace decision reasons
	if _, err := r.db.ExecContext(ctx, `DELETE FROM decision_reasons WHERE assessment_id = $1`, a.ID); err != nil {
		return fmt.Errorf("delete old reasons: %w", err)
	}
	if a.Decision != nil {
		if err := r.insertDecision(ctx, a.ID, *a.Decision); err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
	}

	// Replace evidence
	if _, err := r.db.ExecContext(ctx, `DELETE FROM evidence WHERE assessment_id = $1`, a.ID); err != nil {
		return fmt.Errorf("delete old evidence: %w", err)
	}
	if err := r.insertEvidence(ctx, a.ID, a.Evidence); err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}

	return nil
}

// List retrieves assessments with basic pagination.
// If limit and offset are both 0, all assessments are returned.
func (r *PostgresRepository) List(ctx context.Context, limit, offset int) ([]domain.Assessment, error) {
	query := `
		SELECT id, applicant_id, applicant_name, applicant_age,
			application_id, application_requested_amount, application_purpose,
			score, status, policy_id, policy_version,
			decision_outcome, decision_outputs, decision_policy_id, decision_policy_version,
			error, created_at, started_at, completed_at
		FROM assessments`

	var args []interface{}
	argIdx := 1

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
		argIdx++
	}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list assessments: %w", err)
	}
	defer rows.Close()

	var assessments []domain.Assessment
	for rows.Next() {
		var a domain.Assessment
		var status string
		var scoreJSON sql.NullString
		var outputsJSON sql.NullString
		var errorStr sql.NullString
		var appID, appPurpose sql.NullString
		var appAmount sql.NullInt64
		var decisionOutcome sql.NullString
		var decisionPolicyID string
		var decisionPolicyVersion int
		var createdAt, startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&a.ID, &a.Applicant.ID, &a.Applicant.Name, &a.Applicant.Age,
			&appID, &appAmount, &appPurpose,
			&scoreJSON, &status, &a.PolicyID, &a.PolicyVersion,
			&decisionOutcome, &outputsJSON, &decisionPolicyID, &decisionPolicyVersion,
			&errorStr, &createdAt, &startedAt, &completedAt,
		); err != nil {
			return nil, fmt.Errorf("scan assessment: %w", err)
		}

		// Reconstruct optional Application
		if appID.Valid && appID.String != "" {
			app := &domain.Application{
				ID:      appID.String,
				Purpose: appPurpose.String,
			}
			if appAmount.Valid {
				app.RequestedAmount = &appAmount.Int64
			}
			a.Application = app
		}

		a.Status = parseStatus(status)
		if errorStr.Valid {
			a.Error = errorStr.String
		}
		if createdAt.Valid {
			a.CreatedAt = createdAt.Time
		}
		if startedAt.Valid {
			a.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			a.CompletedAt = &completedAt.Time
		}

		// Reconstruct Score
		if scoreJSON.Valid && scoreJSON.String != "" {
			var score domain.CreditScore
			if err := json.Unmarshal([]byte(scoreJSON.String), &score); err != nil {
				return nil, fmt.Errorf("unmarshal score: %w", err)
			}
			a.Score = &score
		}

		// Reconstruct Decision
		if decisionOutcome.Valid && decisionOutcome.String != "" {
			d := &domain.Decision{
				Outcome:       parseDecisionOutcome(decisionOutcome.String),
				PolicyID:      decisionPolicyID,
				PolicyVersion: decisionPolicyVersion,
			}

			if outputsJSON.Valid && outputsJSON.String != "" {
				var outputs domain.DecisionOutputs
				if err := json.Unmarshal([]byte(outputsJSON.String), &outputs); err != nil {
					return nil, fmt.Errorf("unmarshal outputs: %w", err)
				}
				d.Outputs = &outputs
			}

			a.Decision = d
		}

		assessments = append(assessments, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list iterations: %w", err)
	}

	return assessments, nil
}

func (r *PostgresRepository) insertDecision(ctx context.Context, assessmentID string, d domain.Decision) error {
	for _, reason := range d.Reasons {
		valueJSON, _ := json.Marshal(reason.Value)
		thresholdJSON, _ := json.Marshal(reason.Threshold)

		_, err := r.db.ExecContext(ctx, `
			INSERT INTO decision_reasons (assessment_id, code, description, value, threshold, evidence_ref)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			assessmentID, reason.Code, reason.Description, valueJSON, thresholdJSON, reason.EvidenceRef,
		)
		if err != nil {
			return fmt.Errorf("insert reason %s: %w", reason.Code, err)
		}
	}
	return nil
}

func (r *PostgresRepository) getDecision(ctx context.Context, assessmentID string) ([]domain.DecisionReason, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT code, description, value, threshold, evidence_ref
		FROM decision_reasons WHERE assessment_id = $1`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reasons []domain.DecisionReason
	for rows.Next() {
		var reason domain.DecisionReason
		var valueJSON, thresholdJSON []byte
		if err := rows.Scan(&reason.Code, &reason.Description, &valueJSON, &thresholdJSON, &reason.EvidenceRef); err != nil {
			return nil, err
		}
		if valueJSON != nil {
			_ = json.Unmarshal(valueJSON, &reason.Value)
		}
		if thresholdJSON != nil {
			_ = json.Unmarshal(thresholdJSON, &reason.Threshold)
		}
		reasons = append(reasons, reason)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reasons, nil
}

func (r *PostgresRepository) insertEvidence(ctx context.Context, assessmentID string, evidence []domain.Evidence) error {
	for _, e := range evidence {
		valueJSON, _ := json.Marshal(e.Value)
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO evidence (assessment_id, source, field, value, retrieved_at, reference)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			assessmentID, e.Source, e.Field, valueJSON, e.RetrievedAt, e.Reference,
		)
		if err != nil {
			return fmt.Errorf("insert evidence %s: %w", e.Reference, err)
		}
	}
	return nil
}

func (r *PostgresRepository) getEvidence(ctx context.Context, assessmentID string) ([]domain.Evidence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT source, field, value, retrieved_at, reference
		FROM evidence WHERE assessment_id = $1`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evidence []domain.Evidence
	for rows.Next() {
		var e domain.Evidence
		var valueJSON []byte
		if err := rows.Scan(&e.Source, &e.Field, &valueJSON, &e.RetrievedAt, &e.Reference); err != nil {
			return nil, err
		}
		if valueJSON != nil {
			_ = json.Unmarshal(valueJSON, &e.Value)
		}
		evidence = append(evidence, e)
	}
	return evidence, rows.Err()
}

func jsonScore(s *domain.CreditScore) (sql.NullString, error) {
	if s == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func parseStatus(s string) domain.AssessmentStatus {
	switch s {
	case "PENDING":
		return domain.AssessmentPending
	case "RUNNING":
		return domain.AssessmentRunning
	case "COMPLETED":
		return domain.AssessmentCompleted
	case "FAILED":
		return domain.AssessmentFailed
	default:
		return domain.AssessmentPending
	}
}

func parseDecisionOutcome(s string) domain.DecisionOutcome {
	switch s {
	case "APPROVE":
		return domain.DecisionApprove
	case "REVIEW":
		return domain.DecisionReview
	case "REJECT":
		return domain.DecisionReject
	default:
		return domain.DecisionApprove
	}
}
