package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"credora/internal/domain"
)

func newTestSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := ConnectSQLite(dbPath)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})
	return db
}

func TestSQLiteRepository_CreateAndGet(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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
	if got.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if got.Application.ID != "loan-1" {
		t.Errorf("expected application ID loan-1, got %s", got.Application.ID)
	}
	if got.Application.RequestedAmount == nil || *got.Application.RequestedAmount != 50000 {
		t.Errorf("expected requested amount 50000, got %v", got.Application.RequestedAmount)
	}
	if got.Application.Purpose != "personal" {
		t.Errorf("expected purpose personal, got %s", got.Application.Purpose)
	}
}

func TestSQLiteRepository_CreateDuplicate(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_GetNotFound(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent assessment, got nil")
	}
}

func TestSQLiteRepository_Update(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_UpdateNotFound(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_OptionalApplication(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	a := domain.Assessment{
		ID: "a-no-app",
		Applicant: domain.Applicant{
			ID:   "app-1",
			Name: "Alice",
			Age:  30,
		},
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

func TestSQLiteRepository_ApplicationWithoutRequestedAmount(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	a := domain.Assessment{
		ID: "a-no-amount",
		Applicant: domain.Applicant{
			ID:   "app-1",
			Name: "Bob",
			Age:  25,
		},
		Application: &domain.Application{
			ID:      "loan-1",
			Purpose: "personal",
		},
		Status:    domain.AssessmentCompleted,
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-no-amount")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if got.Application.RequestedAmount != nil {
		t.Error("expected nil RequestedAmount")
	}
}

func TestSQLiteRepository_DecisionPersistence(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_DecisionOutputs(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	creditLimit := int64(75000)
	approvedAmount := int64(50000)

	a := domain.Assessment{
		ID:        "a-outputs",
		Applicant: domain.Applicant{ID: "app-1", Name: "Charlie", Age: 30},
		Status:    domain.AssessmentCompleted,
		PolicyID:  "limit-policy",
		Decision: &domain.Decision{
			Outcome:       domain.DecisionApprove,
			PolicyID:      "limit-policy",
			PolicyVersion: 1,
			Outputs: &domain.DecisionOutputs{
				CreditLimit:    &creditLimit,
				ApprovedAmount: &approvedAmount,
			},
		},
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-outputs")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if got.Decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if got.Decision.Outputs.CreditLimit == nil || *got.Decision.Outputs.CreditLimit != 75000 {
		t.Errorf("expected credit limit 75000, got %v", got.Decision.Outputs.CreditLimit)
	}
	if got.Decision.Outputs.ApprovedAmount == nil || *got.Decision.Outputs.ApprovedAmount != 50000 {
		t.Errorf("expected approved amount 50000, got %v", got.Decision.Outputs.ApprovedAmount)
	}
}

func TestSQLiteRepository_DecisionOutputsUpdate(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	creditLimit := int64(75000)

	a := domain.Assessment{
		ID:        "a-outputs-update",
		Applicant: domain.Applicant{ID: "app-1"},
		Status:    domain.AssessmentRunning,
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	a.Status = domain.AssessmentCompleted
	a.Decision = &domain.Decision{
		Outcome:       domain.DecisionApprove,
		PolicyID:      "limit-policy",
		PolicyVersion: 1,
		Outputs: &domain.DecisionOutputs{
			CreditLimit: &creditLimit,
		},
	}

	if err := repo.Update(ctx, a); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-outputs-update")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if got.Decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if got.Decision.Outputs.CreditLimit == nil || *got.Decision.Outputs.CreditLimit != 75000 {
		t.Errorf("expected credit limit 75000, got %v", got.Decision.Outputs.CreditLimit)
	}
	if got.Decision.Outputs.ApprovedAmount != nil {
		t.Error("expected nil approved amount")
	}
}

func TestSQLiteRepository_DecisionReasons(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now()
	a := domain.Assessment{
		ID:        "a-reasons",
		Applicant: domain.Applicant{ID: "app-1", Name: "Dave", Age: 25},
		Status:    domain.AssessmentCompleted,
		PolicyID:  "test",
		Decision: &domain.Decision{
			Outcome:       domain.DecisionReject,
			PolicyID:      "test",
			PolicyVersion: 1,
			Reasons: []domain.DecisionReason{
				{
					Code:        "LOW_SCORE",
					Description: "Credit score too low",
					Value:       580,
					Threshold:   650,
					EvidenceRef: "SCORE_CHECK",
				},
				{
					Code:        "HIGH_DEBT",
					Description: "Debt ratio too high",
					Value:       0.85,
					Threshold:   0.70,
					EvidenceRef: "DEBT_CHECK",
				},
			},
		},
		Evidence: []domain.Evidence{
			{
				Source:      "credit_bureau",
				Field:       "credit_score",
				Value:       580,
				RetrievedAt: now,
				Reference:   "SCORE_CHECK",
			},
			{
				Source:      "income_provider",
				Field:       "debt_ratio",
				Value:       0.85,
				RetrievedAt: now,
				Reference:   "DEBT_CHECK",
			},
		},
		CreatedAt: now,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-reasons")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if len(got.Decision.Reasons) != 2 {
		t.Fatalf("expected 2 reasons, got %d", len(got.Decision.Reasons))
	}
	if got.Decision.Reasons[0].Code != "LOW_SCORE" {
		t.Errorf("expected first reason LOW_SCORE, got %s", got.Decision.Reasons[0].Code)
	}
	if got.Decision.Reasons[1].Code != "HIGH_DEBT" {
		t.Errorf("expected second reason HIGH_DEBT, got %s", got.Decision.Reasons[1].Code)
	}
	if len(got.Evidence) != 2 {
		t.Errorf("expected 2 evidence, got %d", len(got.Evidence))
	}
}

func TestSQLiteRepository_EvidencePersistence(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now()
	a := domain.Assessment{
		ID:        "a-evidence",
		Applicant: domain.Applicant{ID: "app-1"},
		Status:    domain.AssessmentCompleted,
		CreatedAt: now,
		Evidence: []domain.Evidence{
			{
				Source:      "provider_a",
				Field:       "income",
				Value:       75000,
				RetrievedAt: now,
				Reference:   "inc-001",
			},
			{
				Source:      "provider_b",
				Field:       "employment_length",
				Value:       5,
				RetrievedAt: now,
				Reference:   "emp-001",
			},
			{
				Source:      "provider_c",
				Field:       "address_verified",
				Value:       true,
				RetrievedAt: now,
				Reference:   "addr-001",
			},
		},
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-evidence")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(got.Evidence) != 3 {
		t.Fatalf("expected 3 evidence, got %d", len(got.Evidence))
	}

	sources := make(map[string]bool)
	for _, e := range got.Evidence {
		sources[e.Source] = true
	}
	if !sources["provider_a"] {
		t.Error("expected provider_a evidence")
	}
	if !sources["provider_b"] {
		t.Error("expected provider_b evidence")
	}
	if !sources["provider_c"] {
		t.Error("expected provider_c evidence")
	}
}

func TestSQLiteRepository_PolicyVersionPreserved(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_LifecycleState(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now()
	started := now.Add(time.Second)
	completed := now.Add(2 * time.Second)

	a := domain.Assessment{
		ID:          "a-lifecycle",
		Applicant:   domain.Applicant{ID: "app-1"},
		Status:      domain.AssessmentCompleted,
		CreatedAt:   now,
		StartedAt:   &started,
		CompletedAt: &completed,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-lifecycle")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
	if got.Status != domain.AssessmentCompleted {
		t.Errorf("expected COMPLETED, got %v", got.Status)
	}
}

func TestSQLiteRepository_FailedAssessmentPersistence(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	now := time.Now()
	started := now
	a := domain.Assessment{
		ID:        "a-failed",
		Applicant: domain.Applicant{ID: "app-2"},
		Status:    domain.AssessmentFailed,
		Error:     "policy evaluation failed: missing required fields",
		CreatedAt: now,
		StartedAt: &started,
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-failed")
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

func TestSQLiteRepository_MoneyRepresentation(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	// Verify no float64 conversion: use large integer amounts
	amount := int64(9999999999) // 10 billion in smallest currency unit
	creditLimit := int64(5000000000)

	a := domain.Assessment{
		ID:        "a-money",
		Applicant: domain.Applicant{ID: "app-1"},
		Application: &domain.Application{
			ID:              "loan-1",
			RequestedAmount: &amount,
			Purpose:         "personal",
		},
		Status: domain.AssessmentCompleted,
		Decision: &domain.Decision{
			Outcome:       domain.DecisionApprove,
			PolicyID:      "test",
			PolicyVersion: 1,
			Outputs: &domain.DecisionOutputs{
				CreditLimit: &creditLimit,
			},
		},
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-money")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if got.Application.RequestedAmount == nil || *got.Application.RequestedAmount != 9999999999 {
		t.Errorf("expected requested amount 9999999999, got %v", got.Application.RequestedAmount)
	}
	if got.Decision == nil || got.Decision.Outputs == nil {
		t.Fatal("expected decision outputs, got nil")
	}
	if got.Decision.Outputs.CreditLimit == nil || *got.Decision.Outputs.CreditLimit != 5000000000 {
		t.Errorf("expected credit limit 5000000000, got %v", got.Decision.Outputs.CreditLimit)
	}
}

func TestSQLiteRepository_UpdatePreservesVersion(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
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

func TestSQLiteRepository_MultipleAssessments(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		a := domain.Assessment{
			ID:        fmt.Sprintf("a%d", i),
			Applicant: domain.Applicant{ID: fmt.Sprintf("app-%d", i)},
			Status:    domain.AssessmentPending,
			CreatedAt: time.Now(),
		}
		if err := repo.Create(ctx, a); err != nil {
			t.Fatalf("create a%d: %v", i, err)
		}
	}

	for i := 0; i < 5; i++ {
		got, err := repo.GetByID(ctx, fmt.Sprintf("a%d", i))
		if err != nil {
			t.Fatalf("get a%d: %v", i, err)
		}
		if got.ID != fmt.Sprintf("a%d", i) {
			t.Errorf("expected ID a%d, got %s", i, got.ID)
		}
	}
}

func TestSQLiteRepository_ScorePersistence(t *testing.T) {
	db := newTestSQLiteDB(t)
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	a := domain.Assessment{
		ID:        "a-score",
		Applicant: domain.Applicant{ID: "app-1"},
		Score: &domain.CreditScore{
			Value:    720,
			Provider: "mock-provider",
		},
		Status:    domain.AssessmentCompleted,
		CreatedAt: time.Now(),
	}

	if err := repo.Create(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByID(ctx, "a-score")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Score == nil {
		t.Fatal("expected score, got nil")
	}
	if got.Score.Value != 720 {
		t.Errorf("expected score 720, got %d", got.Score.Value)
	}
	if got.Score.Provider != "mock-provider" {
		t.Errorf("expected provider mock-provider, got %s", got.Score.Provider)
	}
}
