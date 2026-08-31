package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"credora/internal/assessment"
	"credora/internal/domain"
	"credora/internal/policy"
	"credora/internal/repository"
)

func ptr[T any](v T) *T { return &v }

func setupTestHandler(t *testing.T) (*Handler, http.Handler) {
	t.Helper()
	repo := repository.NewMemoryRepository()
	logger := slog.Default()

	registry := policy.NewRegistry()
	_ = registry.Register(domain.Policy{
		ID:      "personal-loan",
		Version: 1,
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
			{
				Code:        "LOW_CREDIT_SCORE",
				Description: "Credit score below 650 is rejected",
				Condition: func(a domain.Assessment) domain.ConditionResult {
					if a.Score == nil {
						return domain.ConditionResult{Matched: false, Detail: "no score provided"}
					}
					return domain.ConditionResult{
						Actual:    a.Score.Value,
						Threshold: 650,
						Matched:   a.Score.Value < 650,
						Detail:    "score < 650",
					}
				},
				Outcome:    domain.RuleOutcomeReject,
				ReasonCode: "LOW_CREDIT_SCORE",
				ReasonDesc: "Credit score is below the minimum threshold",
			},
		},
		ScoreThresholds: &domain.ScoreThresholds{
			ApproveThreshold: 700,
			ReviewThreshold:  500,
		},
		DefaultOutcome: domain.DecisionReview,
	})

	svc := assessment.NewService(repo, registry, logger)
	policyRepo := repository.NewMemoryPolicyRepository()
	handler := NewHandler(svc, policyRepo, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	handler.RegisterRoutes(mux)
	return handler, mux
}

func TestCreateAssessment_Success(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {
			"id": "app-001",
			"name": "Alice",
			"age": 35
		},
		"application": {
			"id": "loan-001",
			"requested_amount": 50000,
			"purpose": "working_capital"
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

	if resp.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", resp.Status)
	}
	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Decision.Outcome)
	}
	if resp.Policy.ID != "personal-loan" {
		t.Errorf("expected policy personal-loan, got %s", resp.Policy.ID)
	}
	if resp.Policy.Version != 1 {
		t.Errorf("expected policy version 1, got %d", resp.Policy.Version)
	}
	if resp.Applicant.ID != "app-001" {
		t.Errorf("expected applicant ID app-001, got %s", resp.Applicant.ID)
	}
	if resp.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if resp.Application.ID != "loan-001" {
		t.Errorf("expected application ID loan-001, got %s", resp.Application.ID)
	}
	if resp.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestCreateAssessment_WithScore(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {
			"id": "app-002",
			"name": "Bob",
			"age": 28
		},
		"application": {
			"id": "loan-002",
			"requested_amount": 100000,
			"purpose": "personal"
		},
		"score": {
			"value": 720,
			"provider": "mock"
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

	if resp.Score == nil {
		t.Fatal("expected score, got nil")
	}
	if resp.Score.Value != 720 {
		t.Errorf("expected score 720, got %d", resp.Score.Value)
	}
}

func TestCreateAssessment_Knockout(t *testing.T) {
	repo := repository.NewMemoryRepository()
	logger := slog.Default()

	registry := policy.NewRegistry()
	_ = registry.Register(domain.Policy{
		ID:      "test-knockout",
		Version: 1,
		Knockouts: []domain.Knockout{
			{
				Code:        "AGE_BELOW_MINIMUM",
				Description: "Applicant is under 18",
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

	svc := assessment.NewService(repo, registry, logger)
	policyRepo := repository.NewMemoryPolicyRepository()
	handler := NewHandler(svc, policyRepo, logger)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := `{
		"applicant": {
			"id": "app-003",
			"name": "Charlie",
			"age": 16
		},
		"application": {
			"id": "loan-003",
			"requested_amount": 50000,
			"purpose": "personal"
		},
		"policy": {
			"id": "test-knockout",
			"version": 1
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

	if resp.Decision == nil {
		t.Fatal("expected decision, got nil")
	}
	if resp.Decision.Outcome != "REJECT" {
		t.Errorf("expected REJECT, got %s", resp.Decision.Outcome)
	}
	if len(resp.Decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(resp.Decision.Reasons))
	}
	if resp.Decision.Reasons[0].Code != "AGE_BELOW_MINIMUM" {
		t.Errorf("expected reason code AGE_BELOW_MINIMUM, got %s", resp.Decision.Reasons[0].Code)
	}
}

func TestCreateAssessment_InvalidJSON(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "INVALID_REQUEST" {
		t.Errorf("expected code INVALID_REQUEST, got %s", resp.Code)
	}
}

func TestCreateAssessment_MissingApplicantID(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {
			"name": "Alice",
			"age": 35
		},
		"application": {
			"id": "loan-001",
			"requested_amount": 5000000,
			"purpose": "working_capital"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestGetAssessment_Success(t *testing.T) {
	_, mux := setupTestHandler(t)

	createBody := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 35},
		"application": {"id": "loan-001", "requested_amount": 5000000, "purpose": "working_capital"}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp AssessmentResponse
	_ = json.NewDecoder(createW.Body).Decode(&createResp)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", getW.Code, getW.Body.String())
	}

	var getResp AssessmentResponse
	if err := json.NewDecoder(getW.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if getResp.ID != createResp.ID {
		t.Errorf("expected ID %s, got %s", createResp.ID, getResp.ID)
	}
	if getResp.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", getResp.Status)
	}
}

func TestGetAssessment_NotFound(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/nonexistent", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if resp.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %s", resp.Code)
	}
}

func TestGetDecision_Success(t *testing.T) {
	_, mux := setupTestHandler(t)

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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/decision", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp DecisionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Outcome != "APPROVE" {
		t.Errorf("expected APPROVE, got %s", resp.Outcome)
	}
	if resp.Policy.ID != "personal-loan" {
		t.Errorf("expected policy personal-loan, got %s", resp.Policy.ID)
	}
}

func TestGetDecision_NotFound(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/nonexistent/decision", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetEvidence_Success(t *testing.T) {
	_, mux := setupTestHandler(t)

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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID+"/evidence", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []EvidenceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp) != 4 {
		t.Fatalf("expected 4 evidence entries, got %d", len(resp))
	}

	sources := make(map[string]bool)
	for _, e := range resp {
		sources[e.Source] = true
	}
	if !sources["knockout"] {
		t.Error("expected knockout evidence")
	}
	if !sources["rule"] {
		t.Error("expected rule evidence")
	}
}

func TestGetEvidence_NotFound(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/nonexistent/evidence", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreateAssessment_EmptyBody(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAssessment_WithDecisionAndEvidence(t *testing.T) {
	_, mux := setupTestHandler(t)

	createBody := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 30},
		"application": {"id": "loan-001", "requested_amount": 200000, "purpose": "business"}
	}`
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createW := httptest.NewRecorder()
	mux.ServeHTTP(createW, createReq)

	var createResp AssessmentResponse
	_ = json.NewDecoder(createW.Body).Decode(&createResp)

	if createResp.Decision == nil {
		t.Fatal("expected decision in create response")
	}
	if createResp.Decision.Outcome != "REVIEW" {
		t.Errorf("expected REVIEW outcome, got %s", createResp.Decision.Outcome)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/assessments/"+createResp.ID, nil)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	var getResp AssessmentResponse
	_ = json.NewDecoder(getW.Body).Decode(&getResp)

	if getResp.Decision == nil {
		t.Fatal("expected decision in get response")
	}
	if len(getResp.Decision.Reasons) != 1 {
		t.Fatalf("expected 1 reason, got %d", len(getResp.Decision.Reasons))
	}
	if getResp.Decision.Reasons[0].Code != "HIGH_REQUEST_AMOUNT" {
		t.Errorf("expected reason HIGH_REQUEST_AMOUNT, got %s", getResp.Decision.Reasons[0].Code)
	}
}

func TestCreateAssessment_LowScoreRejects(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {"id": "app-001", "name": "Alice", "age": 30},
		"application": {"id": "loan-001", "requested_amount": 50000, "purpose": "personal"},
		"score": {"value": 600, "provider": "mock"}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/assessments", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp AssessmentResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.Decision == nil {
		t.Fatal("expected decision")
	}
	if resp.Decision.Outcome != "REJECT" {
		t.Errorf("expected REJECT for score 600, got %s", resp.Decision.Outcome)
	}
}

func TestHealthEndpoint(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestCreateAssessment_ViaMux_Routing(t *testing.T) {
	_, mux := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/wrong/path", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Error("expected non-200 for wrong path")
	}
}

func TestCreateAssessment_WithoutApplication(t *testing.T) {
	_, mux := setupTestHandler(t)

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
}

func TestCreateAssessment_WithApplicationNoAmount(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {
			"id": "app-no-amount",
			"name": "Test",
			"age": 30
		},
		"application": {
			"id": "loan-no-amount",
			"purpose": "personal"
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

	if resp.Application == nil {
		t.Fatal("expected application, got nil")
	}
	if resp.Application.RequestedAmount != nil {
		t.Error("expected nil requested_amount")
	}
}

func TestCreateAssessment_WithOutputs(t *testing.T) {
	_, mux := setupTestHandler(t)

	body := `{
		"applicant": {
			"id": "app-001",
			"name": "Alice",
			"age": 35
		},
		"application": {
			"id": "loan-001",
			"requested_amount": 50000,
			"purpose": "working_capital"
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
	_ = json.NewDecoder(w.Body).Decode(&resp)

	// Outputs should be nil when policy doesn't produce them
	if resp.Decision != nil && resp.Decision.Outputs != nil {
		t.Error("expected nil outputs for default policy")
	}
}

var _ = strings.Contains

// TestOpenAPIConsistency verifies that the canonical OpenAPI spec
// (docs/openapi.yaml) matches the derived runtime spec
// (services/engine/internal/http/openapi.yaml embedded via //go:embed).
// This test ensures the two files are in sync and will catch
// drift if either file is manually edited without updating the other.
func TestOpenAPIConsistency(t *testing.T) {
	canonical, err := os.ReadFile("../../../docs/openapi.yaml")
	if err != nil {
		t.Skip("docs/openapi.yaml not found, skipping consistency check")
	}

	derived, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Skip("services/engine/internal/http/openapi.yaml not found, skipping consistency check")
	}

	if string(canonical) != string(derived) {
		t.Errorf("OpenAPI spec mismatch:\ncanonical (docs/openapi.yaml): %d bytes\nderived (openapi.yaml): %d bytes",
			len(canonical), len(derived))
	}
}
