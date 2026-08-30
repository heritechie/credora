// Package assessment implements the application service for credit assessments.
//
// The service orchestrates the assessment lifecycle:
//   - Create and persist a new assessment
//   - Execute policy evaluation
//   - Persist the decision and evidence
//   - Handle failures with structured errors
//
// The service depends on repository interfaces, not concrete implementations.
// Policy evaluation is delegated to the policy evaluator, which remains
// independent of HTTP and persistence.
package assessment

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"credora/internal/domain"
	"credora/internal/policy"
	"credora/internal/repository"
)

// Service provides assessment lifecycle operations.
type Service struct {
	repo           repository.AssessmentRepository
	policyRegistry *policy.Registry
	logger         *slog.Logger
}

// NewService creates a new assessment service.
func NewService(repo repository.AssessmentRepository, registry *policy.Registry, logger *slog.Logger) *Service {
	return &Service{
		repo:           repo,
		policyRegistry: registry,
		logger:         logger,
	}
}

// CreateRequest contains the minimum data needed to create an assessment.
// Application is optional: not all assessment types have an application context.
// Amount is optional: not all assessment types require a requested amount.
// All monetary values use the smallest currency unit (e.g., cents, sen).
type CreateRequest struct {
	ApplicantID   string
	ApplicantName string
	ApplicantAge  int
	ApplicationID *string // optional
	Amount        *int64  // optional, requested amount in smallest currency unit
	Purpose       *string // optional
	ScoreValue    *int
	ScoreProvider *string
	PolicyID      string // policy ID to use (resolved by caller)
	PolicyVersion int    // policy version to use (resolved by caller)
}

// Create creates a new assessment and evaluates it against the specified policy.
// The policy is resolved from the registry using the PolicyID and PolicyVersion
// fields in the request. The full lifecycle is executed: PENDING → RUNNING → COMPLETED/FAILED.
func (s *Service) Create(ctx context.Context, req CreateRequest) (domain.Assessment, error) {
	pol, ok := s.policyRegistry.Get(req.PolicyID, req.PolicyVersion)
	if !ok {
		return domain.Assessment{}, fmt.Errorf("policy %s v%d not found", req.PolicyID, req.PolicyVersion)
	}

	now := time.Now()

	assessment := domain.Assessment{
		ID: generateID(),
		Applicant: domain.Applicant{
			ID:   req.ApplicantID,
			Name: req.ApplicantName,
			Age:  req.ApplicantAge,
		},
		Status:        domain.AssessmentPending,
		PolicyID:      pol.ID,
		PolicyVersion: pol.Version,
		CreatedAt:     now,
	}

	// Application is optional
	if req.ApplicationID != nil {
		app := domain.Application{
			ID: *req.ApplicationID,
		}
		if req.Amount != nil {
			app.RequestedAmount = req.Amount
		}
		if req.Purpose != nil {
			app.Purpose = *req.Purpose
		}
		assessment.Application = &app
	}

	if req.ScoreValue != nil && req.ScoreProvider != nil {
		assessment.Score = &domain.CreditScore{
			Value:    *req.ScoreValue,
			Provider: *req.ScoreProvider,
		}
	}

	s.logger.Info("assessment created",
		"assessment_id", assessment.ID,
		"policy_id", pol.ID,
		"policy_version", pol.Version,
	)

	if err := s.repo.Create(ctx, assessment); err != nil {
		return domain.Assessment{}, fmt.Errorf("persist assessment: %w", err)
	}

	result, err := s.execute(ctx, assessment, pol)
	if err != nil {
		return result, err
	}

	return result, nil
}

// GetByID retrieves an assessment by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (domain.Assessment, error) {
	return s.repo.GetByID(ctx, id)
}

// execute runs the assessment through the policy evaluator and persists results.
func (s *Service) execute(ctx context.Context, assessment domain.Assessment, pol domain.Policy) (domain.Assessment, error) {
	started := time.Now()
	assessment.Status = domain.AssessmentRunning
	assessment.StartedAt = &started

	if err := s.repo.Update(ctx, assessment); err != nil {
		return assessment, fmt.Errorf("update assessment to RUNNING: %w", err)
	}

	s.logger.Info("assessment executing",
		"assessment_id", assessment.ID,
	)

	decision, trace, err := policy.Evaluate(assessment, pol)
	completed := time.Now()

	if err != nil {
		assessment.Status = domain.AssessmentFailed
		assessment.Error = err.Error()
		assessment.CompletedAt = &completed

		s.logger.Error("assessment failed",
			"assessment_id", assessment.ID,
			"error", err,
		)

		if updateErr := s.repo.Update(ctx, assessment); updateErr != nil {
			s.logger.Error("failed to persist assessment failure",
				"assessment_id", assessment.ID,
				"error", updateErr,
			)
		}

		return assessment, fmt.Errorf("evaluate policy: %w", err)
	}

	assessment.Status = domain.AssessmentCompleted
	assessment.Decision = &decision
	assessment.Evidence = trace.Evidence
	assessment.CompletedAt = &completed

	s.logger.Info("assessment completed",
		"assessment_id", assessment.ID,
		"outcome", decision.Outcome.String(),
		"policy_id", decision.PolicyID,
		"policy_version", decision.PolicyVersion,
	)

	if err := s.repo.Update(ctx, assessment); err != nil {
		return assessment, fmt.Errorf("persist assessment result: %w", err)
	}

	return assessment, nil
}

// generateID produces a random ID suitable for persistence and API references.
// Format: a UUID v4 without hyphens (32 hex characters).
func generateID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	// Set version 4 and variant bits
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x%04x%04x%04x%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
