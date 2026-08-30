package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsEndpoint_Success(t *testing.T) {
	mux := http.NewServeMux()
	RegisterDocsRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected text/html content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "api-reference") {
		t.Error("expected Scalar API reference script tag")
	}
	if !strings.Contains(body, "/openapi.yaml") {
		t.Error("expected reference to /openapi.yaml")
	}
}

func TestOpenAPISpecEndpoint_Success(t *testing.T) {
	mux := http.NewServeMux()
	RegisterDocsRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/yaml") {
		t.Errorf("expected application/yaml content type, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "openapi: 3.1.0") {
		t.Error("expected OpenAPI 3.1.0 version")
	}
	if !strings.Contains(body, "Credora Engine API") {
		t.Error("expected Credora Engine API title")
	}
}

func TestOpenAPISpecEndpoint_ContainsAllEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	RegisterDocsRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()

	endpoints := []string{
		"/health",
		"/api/v1/assessments",
		"/api/v1/assessments/{id}",
		"/api/v1/assessments/{id}/decision",
		"/api/v1/assessments/{id}/evidence",
	}

	for _, endpoint := range endpoints {
		if !strings.Contains(body, endpoint) {
			t.Errorf("expected endpoint %s in spec", endpoint)
		}
	}
}

func TestOpenAPISpecEndpoint_ContainsSchemas(t *testing.T) {
	mux := http.NewServeMux()
	RegisterDocsRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()

	schemas := []string{
		"CreateAssessmentRequest",
		"AssessmentResponse",
		"DecisionResponse",
		"DecisionOutputs",
		"EvidenceEntry",
		"ErrorResponse",
	}

	for _, schema := range schemas {
		if !strings.Contains(body, schema) {
			t.Errorf("expected schema %s in spec", schema)
		}
	}
}

func TestOpenAPISpecEndpoint_OptionalApplication(t *testing.T) {
	mux := http.NewServeMux()
	RegisterDocsRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify application is documented as optional
	if !strings.Contains(body, "Application is Optional") {
		t.Error("expected documentation about optional application")
	}
	if !strings.Contains(body, "requested_amount") {
		t.Error("expected requested_amount field")
	}
}

func TestExistingEndpoints_Unaffected(t *testing.T) {
	_, mux := setupTestHandler(t)

	// Test health endpoint still works
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health endpoint: expected 200, got %d", w.Code)
	}

	// Test create assessment still works
	body := `{"applicant": {"id": "app-001", "name": "Test", "age": 30}}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/assessments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create assessment: expected 201, got %d: %s", w.Code, w.Body.String())
	}
}
