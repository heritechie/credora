package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"credora/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLiteRepository implements AssessmentRepository using SQLite.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite-backed assessment repository.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// ConnectSQLite opens a connection to SQLite using the given DSN.
// The DSN should be a file path (e.g., "./data/credora.db").
func ConnectSQLite(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	// Enable foreign key enforcement.
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	// Run migrations.
	if err := migrate(db, "sqlite"); err != nil {
		db.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return db, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, a domain.Assessment) error {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Applicant.ID, a.Applicant.Name, a.Applicant.Age,
		appID, appAmount, appPurpose,
		scoreJSON, a.Status.String(), a.PolicyID, a.PolicyVersion,
		decisionOutcome, outputsJSON, decisionPolicyID, decisionPolicyVersion,
		nullString(a.Error), formatSQLiteTime(a.CreatedAt), formatSQLiteTimePtr(a.StartedAt).String, formatSQLiteTimePtr(a.CompletedAt).String,
	)
	if err != nil {
		return fmt.Errorf("insert assessment: %w", err)
	}

	if a.Decision != nil {
		if err := r.insertDecisionReasons(ctx, a.ID, a.Decision.Reasons); err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
	}

	if err := r.insertEvidence(ctx, a.ID, a.Evidence); err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (domain.Assessment, error) {
	var a domain.Assessment
	var status string
	var scoreJSON sql.NullString
	var outputsJSON sql.NullString
	var errorStr sql.NullString
	var createdAt, startedAt, completedAt sql.NullString
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
		FROM assessments WHERE id = ?`, id,
	).Scan(
		&a.ID, &a.Applicant.ID, &a.Applicant.Name, &a.Applicant.Age,
		&appID, &appAmount, &appPurpose,
		&scoreJSON, &status, &a.PolicyID, &a.PolicyVersion,
		&decisionOutcome, &outputsJSON, &decisionPolicyID, &decisionPolicyVersion,
		&errorStr, &createdAt, &startedAt, &completedAt,
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
	if createdAt.Valid && createdAt.String != "" {
		t, err := parseSQLiteTime(createdAt.String)
		if err == nil {
			a.CreatedAt = t
		}
	}
	if startedAt.Valid && startedAt.String != "" {
		t, err := parseSQLiteTime(startedAt.String)
		if err == nil {
			a.StartedAt = &t
		}
	}
	if completedAt.Valid && completedAt.String != "" {
		t, err := parseSQLiteTime(completedAt.String)
		if err == nil {
			a.CompletedAt = &t
		}
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

		reasons, err := r.getDecisionReasons(ctx, id)
		if err != nil {
			return domain.Assessment{}, fmt.Errorf("get decision reasons: %w", err)
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

func (r *SQLiteRepository) Update(ctx context.Context, a domain.Assessment) error {
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
			applicant_id = ?, applicant_name = ?, applicant_age = ?,
			application_id = ?, application_requested_amount = ?, application_purpose = ?,
			score = ?, status = ?, policy_id = ?, policy_version = ?,
			decision_outcome = ?, decision_outputs = ?, decision_policy_id = ?, decision_policy_version = ?,
			error = ?, started_at = ?, completed_at = ?
		WHERE id = ?`,
		a.Applicant.ID, a.Applicant.Name, a.Applicant.Age,
		appID, appAmount, appPurpose,
		scoreJSON, a.Status.String(), a.PolicyID, a.PolicyVersion,
		decisionOutcome, outputsJSON, decisionPolicyID, decisionPolicyVersion,
		nullString(a.Error), formatSQLiteTimePtr(a.StartedAt).String, formatSQLiteTimePtr(a.CompletedAt).String,
		a.ID,
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
	if _, err := r.db.ExecContext(ctx, `DELETE FROM decision_reasons WHERE assessment_id = ?`, a.ID); err != nil {
		return fmt.Errorf("delete old reasons: %w", err)
	}
	if a.Decision != nil {
		if err := r.insertDecisionReasons(ctx, a.ID, a.Decision.Reasons); err != nil {
			return fmt.Errorf("insert decision: %w", err)
		}
	}

	// Replace evidence
	if _, err := r.db.ExecContext(ctx, `DELETE FROM evidence WHERE assessment_id = ?`, a.ID); err != nil {
		return fmt.Errorf("delete old evidence: %w", err)
	}
	if err := r.insertEvidence(ctx, a.ID, a.Evidence); err != nil {
		return fmt.Errorf("insert evidence: %w", err)
	}

	return nil
}

func (r *SQLiteRepository) insertDecisionReasons(ctx context.Context, assessmentID string, reasons []domain.DecisionReason) error {
	for _, reason := range reasons {
		valueJSON, _ := json.Marshal(reason.Value)
		thresholdJSON, _ := json.Marshal(reason.Threshold)

		_, err := r.db.ExecContext(ctx, `
			INSERT INTO decision_reasons (assessment_id, code, description, value, threshold, evidence_ref)
			VALUES (?, ?, ?, ?, ?, ?)`,
			assessmentID, reason.Code, reason.Description, valueJSON, thresholdJSON, reason.EvidenceRef,
		)
		if err != nil {
			return fmt.Errorf("insert reason %s: %w", reason.Code, err)
		}
	}
	return nil
}

func (r *SQLiteRepository) getDecisionReasons(ctx context.Context, assessmentID string) ([]domain.DecisionReason, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT code, description, value, threshold, evidence_ref
		FROM decision_reasons WHERE assessment_id = ?`, assessmentID)
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

func (r *SQLiteRepository) insertEvidence(ctx context.Context, assessmentID string, evidence []domain.Evidence) error {
	for _, e := range evidence {
		valueJSON, _ := json.Marshal(e.Value)
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO evidence (assessment_id, source, field, value, retrieved_at, reference)
			VALUES (?, ?, ?, ?, ?, ?)`,
			assessmentID, e.Source, e.Field, valueJSON, formatSQLiteTime(e.RetrievedAt), e.Reference,
		)
		if err != nil {
			return fmt.Errorf("insert evidence %s: %w", e.Reference, err)
		}
	}
	return nil
}

func (r *SQLiteRepository) getEvidence(ctx context.Context, assessmentID string) ([]domain.Evidence, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT source, field, value, retrieved_at, reference
		FROM evidence WHERE assessment_id = ?`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evidence []domain.Evidence
	for rows.Next() {
		var e domain.Evidence
		var valueJSON []byte
		var retrievedAt string
		if err := rows.Scan(&e.Source, &e.Field, &valueJSON, &retrievedAt, &e.Reference); err != nil {
			return nil, err
		}
		if valueJSON != nil {
			_ = json.Unmarshal(valueJSON, &e.Value)
		}
		t, err := parseSQLiteTime(retrievedAt)
		if err == nil {
			e.RetrievedAt = t
		}
		evidence = append(evidence, e)
	}
	return evidence, rows.Err()
}

func jsonOutputs(d *domain.Decision) (sql.NullString, error) {
	if d == nil || d.Outputs == nil {
		return sql.NullString{}, nil
	}
	b, err := json.Marshal(d.Outputs)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: string(b), Valid: true}, nil
}

const sqliteTimeFormat = "2006-01-02T15:04:05.000000Z07:00"

func formatSQLiteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(sqliteTimeFormat)
}

func formatSQLiteTimePtr(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.Format(sqliteTimeFormat), Valid: true}
}

func parseSQLiteTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}
	t, err := time.Parse(sqliteTimeFormat, s)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", s)
	}
	return t, err
}
