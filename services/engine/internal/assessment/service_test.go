package assessment

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"credora/internal/domain"
	"credora/internal/policy"
	"credora/internal/repository"
)

func ptr[T any](v T) *T { return &v }

func testRegistry() *policy.Registry {
	reg := policy.NewRegistry()
	pol := domain.Policy{
		ID:      "test-policy",
		Version: 1,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Applicant must be at least 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
			{
				Code:        "HIGH_AMOUNT",
				Description: "Requests over 100000 need review",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Application == nil || a.Application.RequestedAmount == nil {
						return domain.ConditionResult{Matched: false, Detail: "no application or requested amount"}
					}
					return domain.ConditionResult{
						Actual:    *a.Application.RequestedAmount,
						Threshold: int64(100000),
						Matched:   *a.Application.RequestedAmount > 100000,
						Detail:    "requested_amount > 100000",
					}
				},
				Outcome:    domain.RuleOutcomeReview,
				ReasonCode: "HIGH_REQUEST_AMOUNT",
				ReasonDesc: "Requested amount exceeds review threshold",
			},
		},
	}
	_ = reg.Register(pol)
	return reg
}

func TestService_Create_Success(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-1",
		ApplicantName: "Alice",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-1"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Status != domain.AssessmentCompleted {
		t.Errorf("expected COMPLETED, got %v", a.Status)
	}
	if a.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if a.Decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected APPROVE, got %v", a.Decision.Outcome)
	}
	if a.PolicyID != "test-policy" {
		t.Errorf("expected policy test-policy, got %s", a.PolicyID)
	}
	if a.PolicyVersion != 1 {
		t.Errorf("expected policy version 1, got %d", a.PolicyVersion)
	}
	if a.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
	if a.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestService_Create_Knockout(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := policy.NewRegistry()
	_ = reg.Register(domain.Policy{
		ID:      "ko-policy",
		Version: 1,
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Under 18",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Applicant is under 18",
			},
		},
	})
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-2",
		ApplicantName: "Bob",
		ApplicantAge:  16,
		ApplicationID: ptr("loan-2"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "ko-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Status != domain.AssessmentCompleted {
		t.Errorf("expected COMPLETED, got %v", a.Status)
	}
	if a.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if a.Decision.Outcome != domain.DecisionReject {
		t.Errorf("expected REJECT, got %v", a.Decision.Outcome)
	}
}

func TestService_Create_Review(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-3",
		ApplicantName: "Carol",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-3"),
		Amount:        ptr(int64(200000)),
		Purpose:       ptr("business"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Decision.Outcome != domain.DecisionReview {
		t.Errorf("expected REVIEW, got %v", a.Decision.Outcome)
	}
}

func TestService_Create_WithScore(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	scoreVal := 720
	scoreProvider := "mock"
	req := CreateRequest{
		ApplicantID:   "app-4",
		ApplicantName: "Dave",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-4"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		ScoreValue:    &scoreVal,
		ScoreProvider: &scoreProvider,
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Score == nil {
		t.Fatal("expected score, got nil")
	}
	if a.Score.Value != 720 {
		t.Errorf("expected score 720, got %d", a.Score.Value)
	}
}

func TestService_Create_PersistsResult(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-5",
		ApplicantName: "Eve",
		ApplicantAge:  25,
		ApplicationID: ptr("loan-5"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := repo.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("get persisted: %v", err)
	}

	if got.Status != domain.AssessmentCompleted {
		t.Errorf("expected persisted status COMPLETED, got %v", got.Status)
	}
	if got.Decision == nil {
		t.Fatal("expected persisted decision, got nil")
	}
	if got.Decision.Outcome != domain.DecisionApprove {
		t.Errorf("expected persisted APPROVE, got %v", got.Decision.Outcome)
	}
}

func TestService_Create_EvidencePersisted(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-6",
		ApplicantName: "Frank",
		ApplicantAge:  16,
		ApplicationID: ptr("loan-6"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(a.Evidence) == 0 {
		t.Error("expected evidence entries, got none")
	}

	got, err := repo.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("get persisted: %v", err)
	}
	if len(got.Evidence) == 0 {
		t.Error("expected persisted evidence, got none")
	}
}

func TestService_Create_PolicyVersionPreserved(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := policy.NewRegistry()
	_ = reg.Register(domain.Policy{
		ID:      "loan-policy",
		Version: 3,
		Rules: []domain.Rule{
			{
				Code:        "AGE_MINIMUM",
				Description: "Must be 18+",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					return domain.ConditionResult{
						Actual:    a.Applicant.Age,
						Threshold: 18,
						Matched:   a.Applicant.Age < 18,
						Detail:    "age < 18",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "AGE_BELOW_MINIMUM",
				ReasonDesc: "Under 18",
			},
		},
	})
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-7",
		ApplicantName: "Grace",
		ApplicantAge:  25,
		ApplicationID: ptr("loan-7"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "loan-policy",
		PolicyVersion: 3,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.PolicyID != "loan-policy" {
		t.Errorf("expected policy ID loan-policy, got %s", a.PolicyID)
	}
	if a.PolicyVersion != 3 {
		t.Errorf("expected policy version 3, got %d", a.PolicyVersion)
	}
	if a.Decision.PolicyID != "loan-policy" {
		t.Errorf("expected decision policy ID loan-policy, got %s", a.Decision.PolicyID)
	}
	if a.Decision.PolicyVersion != 3 {
		t.Errorf("expected decision policy version 3, got %d", a.Decision.PolicyVersion)
	}
}

func TestService_GetByID_Success(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-8",
		ApplicantName: "Hank",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-8"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := svc.GetByID(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.ID != a.ID {
		t.Errorf("expected ID %s, got %s", a.ID, got.ID)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := policy.NewRegistry()
	svc := NewService(repo, reg, slog.Default())

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_Create_ValidationFails(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "",
		ApplicantName: "Test",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-1"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty applicant ID, got nil")
	}

	if !strings.Contains(err.Error(), "applicant ID is required") {
		t.Errorf("expected error about applicant ID, got: %v", err)
	}
}

func TestService_Create_TimestampsSet(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	before := time.Now()
	req := CreateRequest{
		ApplicantID:   "app-9",
		ApplicantName: "Ivy",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-9"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	after := time.Now()

	if a.CreatedAt.Before(before) || a.CreatedAt.After(after) {
		t.Error("created_at out of expected range")
	}
	if a.StartedAt == nil {
		t.Error("expected started_at to be set")
	}
	if a.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestService_Create_Deterministic(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-10",
		ApplicantName: "Jane",
		ApplicantAge:  25,
		ApplicationID: ptr("loan-10"),
		Amount:        ptr(int64(50000)),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a1, _ := svc.Create(context.Background(), req)
	a2, _ := svc.Create(context.Background(), req)

	if a1.Decision.Outcome != a2.Decision.Outcome {
		t.Errorf("non-deterministic: %v != %v", a1.Decision.Outcome, a2.Decision.Outcome)
	}
}

func TestService_Create_WithoutApplication(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := policy.NewRegistry()
	_ = reg.Register(domain.Policy{
		ID:             "limit-policy",
		Version:        1,
		DefaultOutcome: domain.DecisionReview,
	})
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-11",
		ApplicantName: "Kate",
		ApplicantAge:  30,
		PolicyID:      "limit-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Application != nil {
		t.Error("expected nil application")
	}
	if a.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
}

func TestService_Create_WithoutRequestedAmount(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := testRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-12",
		ApplicantName: "Leo",
		ApplicantAge:  30,
		ApplicationID: ptr("loan-12"),
		Purpose:       ptr("personal"),
		PolicyID:      "test-policy",
		PolicyVersion: 1,
	}

	a, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if a.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if a.Application.RequestedAmount != nil {
		t.Error("expected nil RequestedAmount")
	}
	if a.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
}

func TestService_Create_PolicyNotFound(t *testing.T) {
	repo := repository.NewMemoryRepository()
	reg := policy.NewRegistry()
	svc := NewService(repo, reg, slog.Default())

	req := CreateRequest{
		ApplicantID:   "app-13",
		ApplicantName: "Mia",
		ApplicantAge:  30,
		PolicyID:      "nonexistent",
		PolicyVersion: 1,
	}

	_, err := svc.Create(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent policy, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected error about policy not found, got: %v", err)
	}
}
