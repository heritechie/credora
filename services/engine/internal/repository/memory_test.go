package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"credora/internal/domain"
)

func ptr[T any](v T) *T { return &v }

func TestMemoryRepository_CreateAndGet(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	a := domain.Assessment{
		ID: "a1",
		Applicant: domain.Applicant{
			ID:   "app-1",
			Name: "Alice",
			Age:  30,
		},
		Application: &domain.Application{
			ID:              "loan-1",
			RequestedAmount: ptr(int64(50000)),
			Purpose:         "personal",
		},
		Status:    domain.AssessmentPending,
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != "a1" {
		t.Errorf("expected ID a1, got %s", got.ID)
	}
	if got.Applicant.Name != "Alice" {
		t.Errorf("expected applicant name Alice, got %s", got.Applicant.Name)
	}
	if got.Status != domain.AssessmentPending {
		t.Errorf("expected status PENDING, got %v", got.Status)
	}
}

func TestMemoryRepository_CreateDuplicate(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	a := domain.Assessment{ID: "a1", Status: domain.AssessmentPending, CreatedAt: time.Now()}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("first create: %v", err)
	}

	err := repo.Create(ctx, a)
	if err == nil {
		t.Fatal("expected error for duplicate create, got nil")
	}
}

func TestMemoryRepository_GetNotFound(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent assessment, got nil")
	}
}

func TestMemoryRepository_Update(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	a := domain.Assessment{
		ID:        "a1",
		Status:    domain.AssessmentPending,
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	a.Status = domain.AssessmentRunning
	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetByID(ctx, "a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domain.AssessmentRunning {
		t.Errorf("expected status RUNNING, got %v", got.Status)
	}
}

func TestMemoryRepository_UpdateNotFound(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	a := domain.Assessment{
		ID:        "nonexistent",
		Status:    domain.AssessmentPending,
		CreatedAt: time.Now(),
	}

	err := repo.Update(ctx, a)
	if err == nil {
		t.Fatal("expected error for update of nonexistent assessment, got nil")
	}
}

func TestMemoryRepository_DecisionPersistence(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()
	a := domain.Assessment{
		ID:            "a1",
		Applicant:     domain.Applicant{ID: "app-1", Name: "Bob", Age: 25},
		Application:   &domain.Application{ID: "loan-1", RequestedAmount: ptr(int64(100000)), Purpose: "business"},
		Status:        domain.AssessmentCompleted,
		PolicyID:      "test-policy",
		PolicyVersion: 2,
		Decision: &domain.Decision{
			Outcome:       domain.DecisionApprove,
			PolicyID:      "test-policy",
			PolicyVersion: 2,
			Reasons: []domain.DecisionReason{
				{
					Code:        "AGE_OK",
					Description: "Age meets minimum requirement",
					Value:       25,
					Threshold:   18,
					EvidenceRef: "AGE_MIN",
				},
			},
		},
		Evidence: []domain.Evidence{
			{
				Source:      "rule",
				Field:       "AGE_MIN",
				Value:       true,
				RetrievedAt: now,
				Reference:   "AGE_MIN",
			},
		},
		CreatedAt:   now,
		CompletedAt: &now,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if got.Decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", got.Decision.Outcome)
	}
	if len(got.Decision.Reasons) != 1 {
		t.Errorf("expected 1 reason, got %d", len(got.Decision.Reasons))
	}
	if got.Decision.PolicyID != "test-policy" {
		t.Errorf("expected policy ID test-policy, got %s", got.Decision.PolicyID)
	}
	if got.Decision.PolicyVersion != 2 {
		t.Errorf("expected policy version 2, got %d", got.Decision.PolicyVersion)
	}
	if len(got.Evidence) != 1 {
		t.Errorf("expected 1 evidence, got %d", len(got.Evidence))
	}
}

func TestMemoryRepository_FailedAssessmentPersistence(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()
	started := now
	a := domain.Assessment{
		ID:        "a2",
		Applicant: domain.Applicant{ID: "app-2"},
		Status:    domain.AssessmentFailed,
		Error:     "policy evaluation failed: missing required fields",
		CreatedAt: now,
		StartedAt: &started,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a2")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Status != domain.AssessmentFailed {
		t.Errorf("expected FAILED, got %v", got.Status)
	}
	if got.Error != "policy evaluation failed: missing required fields" {
		t.Errorf("expected error message, got %q", got.Error)
	}
	if got.Decision != nil {
		t.Error("expected nil decision for failed assessment")
	}
}

func TestMemoryRepository_PolicyVersionPreserved(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()

	a1 := domain.Assessment{
		ID:            "a-v1",
		Applicant:     domain.Applicant{ID: "app-1"},
		Status:        domain.AssessmentCompleted,
		PolicyID:      "loan-policy",
		PolicyVersion: 1,
		Decision: &domain.Decision{
			Outcome:       domain.DecisionApprove,
			PolicyID:      "loan-policy",
			PolicyVersion: 1,
		},
		CreatedAt: now,
	}

	a2 := domain.Assessment{
		ID:            "a-v2",
		Applicant:     domain.Applicant{ID: "app-1"},
		Status:        domain.AssessmentCompleted,
		PolicyID:      "loan-policy",
		PolicyVersion: 2,
		Decision: &domain.Decision{
			Outcome:       domain.DecisionReject,
			PolicyID:      "loan-policy",
			PolicyVersion: 2,
			Reasons: []domain.DecisionReason{
				{Code: "LOW_SCORE", Description: "Score too low"},
			},
		},
		CreatedAt: now,
	}

	if err := repo.Create(ctx, a1); err != nil {
		t.Fatalf("create a1: %v", err)
	}
	if err := repo.Create(ctx, a2); err != nil {
		t.Fatalf("create a2: %v", err)
	}

	got1, err := repo.GetByID(ctx, "a-v1")
	if err != nil {
		t.Fatalf("get a1: %v", err)
	}
	got2, err := repo.GetByID(ctx, "a-v2")
	if err != nil {
		t.Fatalf("get a2: %v", err)
	}

	if got1.Decision.PolicyVersion != 1 {
		t.Errorf("expected a1 policy version 1, got %d", got1.Decision.PolicyVersion)
	}
	if got2.Decision.PolicyVersion != 2 {
		t.Errorf("expected a2 policy version 2, got %d", got2.Decision.PolicyVersion)
	}
	if got1.Decision.Outcome == got2.Decision.Outcome {
		t.Error("expected different outcomes for different policy versions")
	}
}

func TestMemoryRepository_UpdatePreservesVersion(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	now := time.Now()
	a := domain.Assessment{
		ID:            "a1",
		Status:        domain.AssessmentPending,
		PolicyID:      "p1",
		PolicyVersion: 3,
		CreatedAt:     now,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	a.Status = domain.AssessmentCompleted
	decision := domain.Decision{
		Outcome:       domain.DecisionReview,
		PolicyID:      "p1",
		PolicyVersion: 3,
	}
	a.Decision = &decision

	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetByID(ctx, "a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.PolicyVersion != 3 {
		t.Errorf("expected policy version 3, got %d", got.PolicyVersion)
	}
	if got.Decision.PolicyVersion != 3 {
		t.Errorf("expected decision policy version 3, got %d", got.Decision.PolicyVersion)
	}
}

func TestMemoryRepository_ConcurrentAccess(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			a := domain.Assessment{
				ID:        fmt.Sprintf("a%d", n),
				Status:    domain.AssessmentPending,
				CreatedAt: time.Now(),
			}
			done <- repo.Create(ctx, a)
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent create: %v", err)
		}
	}

	for i := 0; i < 10; i++ {
		go func(n int) {
			_, err := repo.GetByID(ctx, fmt.Sprintf("a%d", n))
			done <- err
		}(i)
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent read: %v", err)
		}
	}
}

func TestMemoryRepository_WithoutApplication(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()

	a := domain.Assessment{
		ID: "a-no-app",
		Applicant: domain.Applicant{
			ID:   "app-1",
			Name: "Alice",
			Age:  30,
		},
		// No Application
		Status:    domain.AssessmentCompleted,
		CreatedAt: time.Now(),
		Decision: &domain.Decision{
			Outcome:       domain.DecisionApprove,
			PolicyID:      "limit-policy",
			PolicyVersion: 1,
		},
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-no-app")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Application != nil {
		t.Error("expected nil application")
	}
	if got.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if got.Decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", got.Decision.Outcome)
	}
}
