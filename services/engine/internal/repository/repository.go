// Package repository defines persistence interfaces for the assessment domain.
//
// Repository implementations are adapters: they depend on the domain,
// not the other way around. The domain and application layers depend
// only on these interfaces.
package repository

import (
	"context"

	"credora/internal/domain"
)

// AssessmentRepository defines persistence operations for assessments.
type AssessmentRepository interface {
	Create(ctx context.Context, assessment domain.Assessment) error
	GetByID(ctx context.Context, id string) (domain.Assessment, error)
	Update(ctx context.Context, assessment domain.Assessment) error
}

// PolicyMetadata represents persistable policy metadata.
// The executable policy logic (knockouts, rules, score thresholds) lives in
// the policy registry in Go code. This struct represents what is stored in
// the database for audit and metadata purposes.
type PolicyMetadata struct {
	ID          string
	Version     int
	Name        string
	Description string
}

// PolicyRepository defines persistence operations for policy metadata.
// The executable policy definition is maintained in Go code (policy registry).
// This repository persists policy metadata for audit and metadata purposes.
type PolicyRepository interface {
	// Get retrieves policy metadata by ID and version.
	// Returns an error if the policy is not found.
	Get(ctx context.Context, id string, version int) (PolicyMetadata, error)

	// Save persists policy metadata. It is idempotent: saving the same
	// policy ID and version overwrites the existing metadata.
	Save(ctx context.Context, meta PolicyMetadata) error

	// Exists reports whether a policy with the given ID and version exists.
	Exists(ctx context.Context, id string, version int) (bool, error)
}
