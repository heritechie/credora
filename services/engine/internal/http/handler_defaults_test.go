package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"credora/internal/assessment"
	"credora/internal/policy"
	"credora/internal/repository"
)

// setupDefaultHandler wires a handler using the engine's default policies
// (personal-loan v1), mirroring the production server bootstrap.
func setupDefaultHandler(t *testing.T) http.Handler {
	t.Helper()

	repo := repository.NewMemoryRepository()
	logger := slog.Default()

	registry := policy.NewRegistry()
	if err := policy.RegisterDefaults(registry); err != nil {
		t.Fatalf("register defaults: %v", err)
	}

	policyRepo := repository.NewMemoryPolicyRepository()
	handler := NewHandler(assessment.NewService(repo, registry, logger), policyRepo, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux
}

func postJSON(t *testing.T, mux http.Handler, body string) (*http.Response, AssessmentResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	var resp AssessmentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return w.Result(), resp
}

// TestCreateAssessment_LimitAssessment verifies the default policy produces a
// credit limit for a limit assessment (no application, no requested amount).
func TestCreateAssessment_LimitAssessment(t *testing.T) {
	mux := setupDefaultHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Jane", "age": 35},
		"monthly_income": 10000000,
		"monthly_obligations": 3000000,
		"score": {"value": 720, "provider": "mock-credit-bureau"},
		"policy": {"id": "personal-loan", "version": 1}
	}`

	res, resp := postJSON(t, mux, body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Decision.Outcome)
	}
	if resp.Decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if resp.Decision.Outputs.CreditLimit == nil || *resp.Decision.Outputs.CreditLimit != 20000000 {
		t.Errorf("expected credit limit 20000000, got %v", resp.Decision.Outputs.CreditLimit)
	}
	if resp.Decision.Outputs.ApprovedAmount != nil {
		t.Errorf("expected no approved amount for limit assessment, got %v", resp.Decision.Outputs.ApprovedAmount)
	}
}

// TestCreateAssessment_LoanApplication verifies the default policy approves a
// loan application with a requested amount and produces outputs.
func TestCreateAssessment_LoanApplication(t *testing.T) {
	mux := setupDefaultHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Jane", "age": 35},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "working_capital"},
		"monthly_income": 10000000,
		"monthly_obligations": 3000000,
		"score": {"value": 720, "provider": "mock-credit-bureau"},
		"policy": {"id": "personal-loan", "version": 1}
	}`

	res, resp := postJSON(t, mux, body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Decision.Outcome)
	}
	if resp.Decision.Outputs == nil {
		t.Fatal("expected outputs, got nil")
	}
	if resp.Decision.Outputs.CreditLimit == nil || *resp.Decision.Outputs.CreditLimit != 20000000 {
		t.Errorf("expected credit limit 20000000, got %v", resp.Decision.Outputs.CreditLimit)
	}
	if resp.Decision.Outputs.ApprovedAmount == nil || *resp.Decision.Outputs.ApprovedAmount != 50000 {
		t.Errorf("expected approved amount 50000, got %v", resp.Decision.Outputs.ApprovedAmount)
	}
}

// TestCreateAssessment_HighDSRRejects verifies the HIGH_DSR knockout fires
// through the HTTP API.
func TestCreateAssessment_HighDSRRejects(t *testing.T) {
	mux := setupDefaultHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Jane", "age": 35},
		"monthly_income": 10000000,
		"monthly_obligations": 8000000,
		"score": {"value": 720, "provider": "mock-credit-bureau"},
		"policy": {"id": "personal-loan", "version": 1}
	}`

	res, resp := postJSON(t, mux, body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "REJECT" {
		t.Errorf("expected REJECT, got %s", resp.Decision.Outcome)
	}
	if len(resp.Decision.Reasons) != 1 || resp.Decision.Reasons[0].Code != "HIGH_DSR" {
		t.Errorf("expected HIGH_DSR reason, got %+v", resp.Decision.Reasons)
	}
}

// TestCreateAssessment_NoFinancialsNoPanic verifies an assessment without
// monthly income/obligations evaluates without panicking (financial facts are
// optional inputs to the default policy).
func TestCreateAssessment_NoFinancialsNoPanic(t *testing.T) {
	mux := setupDefaultHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Jane", "age": 35},
		"score": {"value": 720, "provider": "mock-credit-bureau"},
		"policy": {"id": "personal-loan", "version": 1}
	}`

	res, resp := postJSON(t, mux, body)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.StatusCode, res.Body)
	}
	if resp.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", resp.Status)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
}
