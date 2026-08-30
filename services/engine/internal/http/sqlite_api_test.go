package http

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"credora/internal/assessment"
	"credora/internal/domain"
	"credora/internal/policy"
	"credora/internal/repository"
)

func setupSQLiteTestHandler(t *testing.T) (*Handler, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := repository.ConnectSQLite(dbPath)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbPath)
		os.Remove(dbPath + "-wal")
		os.Remove(dbPath + "-shm")
	})

	repo := repository.NewSQLiteRepository(db)
	policyRepo := repository.NewSQLitePolicyRepository(db)
	logger := slog.Default()

	registry := policy.NewRegistry()
	_ = registry.Register(domain.Policy{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan",
		Description: "Default personal loan assessment policy",
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Applicant must be at least 18",
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
				Code:        "HIGH_REQUEST_AMOUNT",
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
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 700,
			ReviewThreshold:  500,
		},
		DefaultOutcome: domain.DecisionReview,
	})

	// Seed policy metadata
	_ = policyRepo.Save(context.Background(), repository.PolicyMetadata{
		ID:          "personal-loan",
		Version:     1,
		Name:        "Personal Loan",
		Description: "Default personal loan assessment policy",
	})

	svc := assessment.NewService(repo, registry, logger)
	handler := NewHandler(svc, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux
}

func TestSQLiteAPI_CreateAssessment_WithoutApplication(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	body := `{
		"applicant": {
			"id": "app-no-app",
			"name": "Test",
			"age": 30
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp AssessmentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Application != nil {
		t.Error("expected nil application")
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Decision.Outcome)
	}
}

func TestSQLiteAPI_CreateAssessment_WithApplication(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 35},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "working_capital"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp AssessmentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if resp.Application.ID != "loan-001" {
		t.Errorf("expected application ID loan-001, got %s", resp.Application.ID)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Decision.Outcome)
	}
}

func TestSQLiteAPI_CreateAssessment_ApplicationWithoutAmount(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	body := `{
		"applicant": {"id": "app-no-amount", "name": "Test", "age": 30},
		"application": {"id": "loan-no-amount", "purpose": "personal"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp AssessmentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if resp.Application.RequestedAmount != nil {
		t.Error("expected nil requested_amount")
	}
}

func TestSQLiteAPI_DecisionWithCreditLimitOutput(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 35},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "working_capital"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createResp AssessmentResponse
	_ = json.NewDecoder(w.Body).Decode(&createResp)

	if createResp.Decision != nil && createResp.Decision.Outputs != nil {
		t.Error("expected nil outputs for default policy")
	}

	// Verify GET works
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var getResp AssessmentResponse
	_ = json.NewDecoder(getW.Body).Decode(&getResp)

	if getResp.Decision == nil {
		t.Fatal("expected decision in get response")
	}
	if getResp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", getResp.Decision.Outcome)
	}
}

func TestSQLiteAPI_DecisionEndpoint(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	createBody := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 35},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "working_capital"}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp AssessmentResponse
	_ = json.NewDecoder(createW.Body).Decode(&createResp)

	// Test decision endpoint
	decReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/decision", nil)
	decW := httptest.NewRecorder()
	mux.ServeHTTP(decW, decReq)

	if decW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", decW.Code, decW.Body.String())
	}

	var decResp DecisionResponse
	if err := json.NewDecoder(decW.Body).Decode(&decResp); err != nil {
		t.Fatalf("decode decision: %v", err)
	}

	if decResp.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", decResp.Outcome)
	}
	if decResp.Policy.ID != "personal-loan" {
		t.Errorf("expected policy personal-loan, got %s", decResp.Policy.ID)
	}
}

func TestSQLiteAPI_EvidenceEndpoint(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	createBody := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 16},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "personal"}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp AssessmentResponse
	_ = json.NewDecoder(createW.Body).Decode(&createResp)

	// Test evidence endpoint
	evReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/evidence", nil)
	evW := httptest.NewRecorder()
	mux.ServeHTTP(evW, evReq)

	if evW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", evW.Code, evW.Body.String())
	}

	var evidence []EvidenceResponse
	if err := json.NewDecoder(evW.Body).Decode(&evidence); err != nil {
		t.Fatalf("decode evidence: %v", err)
	}

	if len(evidence) == 0 {
		t.Error("expected evidence entries, got none")
	}
}

func TestSQLiteAPI_FullLifecycle(t *testing.T) {
	_, mux := setupSQLiteTestHandler(t)

	// Create assessment
	createBody := `{
		"applicant": {"id": "app-lifecycle", "name": "Test", "age": 25},
		"application": {"id": "loan-lifecycle", "requested_amount": 75000, "purpose": "business"}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	if createW.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createW.Code, createW.Body.String())
	}

	var createResp AssessmentResponse
	_ = json.NewDecoder(createW.Body).Decode(&createResp)

	if createResp.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", createResp.Status)
	}
	if createResp.Decision == nil {
		t.Fatal("expected decision")
	}
	if createResp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", createResp.Decision.Outcome)
	}

	// Get assessment
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var getResp AssessmentResponse
	_ = json.NewDecoder(getW.Body).Decode(&getResp)

	if getResp.ID != createResp.ID {
		t.Errorf("expected same ID, got %s vs %s", createResp.ID, getResp.ID)
	}
	if getResp.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED on get, got %s", getResp.Status)
	}
	if getResp.Decision == nil {
		t.Fatal("expected decision on get")
	}

	// Get decision
	decReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/decision", nil)
	decW := httptest.NewRecorder()
	mux.ServeHTTP(decW, decReq)

	if decW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", decW.Code, decW.Body.String())
	}

	// Get evidence
	evReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/evidence", nil)
	evW := httptest.NewRecorder()
	mux.ServeHTTP(evW, evReq)

	if evW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", evW.Code, evW.Body.String())
	}
}
