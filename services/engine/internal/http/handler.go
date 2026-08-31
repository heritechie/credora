package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"credora/internal/assessment"
	"credora/internal/domain"
	"credora/internal/repository"
)

// Handler provides HTTP handlers for the assessment API.
type Handler struct {
	svc        *assessment.Service
	policyRepo repository.PolicyRepository
	logger     *slog.Logger
}

// NewHandler creates a new HTTP handler.
func NewHandler(svc *assessment.Service, policyRepo repository.PolicyRepository, logger *slog.Logger) *Handler {
	return &Handler{
		svc:        svc,
		policyRepo: policyRepo,
		logger:     logger,
	}
}

// RegisterRoutes registers the API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/assessments", h.CreateAssessment)
	mux.HandleFunc("GET /api/v1/assessments/{id}", h.GetAssessment)
	mux.HandleFunc("GET /api/v1/assessments/{id}/decision", h.GetDecision)
	mux.HandleFunc("GET /api/v1/assessments/{id}/evidence", h.GetEvidence)
	mux.HandleFunc("GET /api/v1/assessments", h.GetAssessments)
	mux.HandleFunc("GET /api/v1/policies", h.GetPolicies)
}

// CreateAssessment handles POST /api/v1/assessments.
func (h *Handler) CreateAssessment(w http.ResponseWriter, r *http.Request) {
	var req CreateAssessmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload")
		return
	}

	if err := h.validateCreateRequest(req); err != nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	createReq := assessment.CreateRequest{
		ApplicantID:        req.Applicant.ID,
		ApplicantName:      req.Applicant.Name,
		ApplicantAge:       req.Applicant.Age,
		MonthlyIncome:      req.MonthlyIncome,
		MonthlyObligations: req.MonthlyObligations,
	}

	if req.Application != nil {
		createReq.ApplicationID = &req.Application.ID
		createReq.Amount = req.Application.RequestedAmount
		createReq.Purpose = &req.Application.Purpose
	}

	if req.Score != nil {
		createReq.ScoreValue = &req.Score.Value
		createReq.ScoreProvider = &req.Score.Provider
	}

	// Resolve policy from request, with defaults
	createReq.PolicyID = "personal-loan"
	createReq.PolicyVersion = 1
	if req.Policy != nil {
		if req.Policy.ID != "" {
			createReq.PolicyID = req.Policy.ID
		}
		if req.Policy.Version != 0 {
			createReq.PolicyVersion = req.Policy.Version
		}
	}

	a, err := h.svc.Create(r.Context(), createReq)
	if err != nil {
		h.logger.Error("failed to create assessment",
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "EVALUATION_ERROR", "assessment evaluation failed")
		return
	}

	h.writeJSON(w, http.StatusCreated, h.toAssessmentResponse(a))
}

// GetAssessment handles GET /api/v1/assessments/{id}.
func (h *Handler) GetAssessment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "assessment ID is required")
		return
	}

	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "assessment not found")
			return
		}
		h.logger.Error("failed to get assessment",
			"assessment_id", id,
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve assessment")
		return
	}

	h.writeJSON(w, http.StatusOK, h.toAssessmentResponse(a))
}

// GetDecision handles GET /api/v1/assessments/{id}/decision.
func (h *Handler) GetDecision(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "assessment ID is required")
		return
	}

	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "assessment not found")
			return
		}
		h.logger.Error("failed to get assessment for decision",
			"assessment_id", id,
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve assessment")
		return
	}

	if a.Decision == nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "decision not yet available")
		return
	}

	h.writeJSON(w, http.StatusOK, h.toDecisionResponse(*a.Decision))
}

// GetEvidence handles GET /api/v1/assessments/{id}/evidence.
func (h *Handler) GetEvidence(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "assessment ID is required")
		return
	}

	a, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "assessment not found")
			return
		}
		h.logger.Error("failed to get assessment for evidence",
			"assessment_id", id,
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve assessment")
		return
	}

	evidence := make([]EvidenceResponse, 0, len(a.Evidence))
	for _, e := range a.Evidence {
		evidence = append(evidence, EvidenceResponse{
			Source:      e.Source,
			Field:       e.Field,
			Value:       e.Value,
			RetrievedAt: e.RetrievedAt,
			Reference:   e.Reference,
		})
	}

	h.writeJSON(w, http.StatusOK, evidence)
}

// GetAssessments handles GET /api/v1/assessments.
func (h *Handler) GetAssessments(w http.ResponseWriter, r *http.Request) {
	limit := 0
	offset := 0

	assessments, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		h.logger.Error("failed to list assessments",
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "REPOSITORY_ERROR", "failed to retrieve assessments")
		return
	}

	items := make([]AssessmentListItem, 0, len(assessments))
	for _, a := range assessments {
		item := AssessmentListItem{
			ID:        a.ID,
			Status:    a.Status.String(),
			Policy:    PolicyDTO{ID: a.PolicyID, Version: a.PolicyVersion},
			CreatedAt: a.CreatedAt,
		}

		if a.Decision != nil {
			item.Decision = &DecisionList{
				Outcome: a.Decision.Outcome.String(),
			}
		}

		if a.CompletedAt != nil {
			item.CompletedAt = a.CompletedAt
		}

		items = append(items, item)
	}

	h.writeJSON(w, http.StatusOK, AssessmentListResponse{
		Items: items,
	})
}

// GetPolicies handles GET /api/v1/policies.
func (h *Handler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	metas, err := h.policyRepo.List(r.Context())
	if err != nil {
		h.logger.Error("failed to list policies",
			"error", err,
		)
		h.writeError(w, http.StatusInternalServerError, "REPOSITORY_ERROR", "failed to retrieve policies")
		return
	}

	items := make([]PolicyListItem, 0, len(metas))
	for _, m := range metas {
		items = append(items, PolicyListItem{
			ID:          m.ID,
			Version:     m.Version,
			Description: m.Description,
			Status:      "active",
		})
	}

	h.writeJSON(w, http.StatusOK, PolicyListResponse{
		Items: items,
	})
}

func (h *Handler) validateCreateRequest(req CreateAssessmentRequest) error {
	var errs []string

	if req.Applicant.ID == "" {
		errs = append(errs, "applicant.id is required")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (h *Handler) toAssessmentResponse(a domain.Assessment) AssessmentResponse {
	resp := AssessmentResponse{
		ID:     a.ID,
		Status: a.Status.String(),
		Applicant: ApplicantDTO{
			ID:   a.Applicant.ID,
			Name: a.Applicant.Name,
			Age:  a.Applicant.Age,
		},
		Policy: PolicyDTO{
			ID:      a.PolicyID,
			Version: a.PolicyVersion,
		},
		CreatedAt:   a.CreatedAt,
		StartedAt:   a.StartedAt,
		CompletedAt: a.CompletedAt,
	}

	if a.Application != nil {
		app := ApplicationDTO{
			ID:      a.Application.ID,
			Purpose: a.Application.Purpose,
		}
		if a.Application.RequestedAmount != nil {
			app.RequestedAmount = a.Application.RequestedAmount
		}
		resp.Application = &app
	}

	if a.Score != nil {
		resp.Score = &ScoreDTO{
			Value:    a.Score.Value,
			Provider: a.Score.Provider,
		}
	}

	if a.Decision != nil {
		d := h.toDecisionResponse(*a.Decision)
		resp.Decision = &d
	}

	if a.Error != "" {
		resp.Error = a.Error
	}

	return resp
}

func (h *Handler) toDecisionResponse(d domain.Decision) DecisionResponse {
	reasons := make([]ReasonResponse, 0, len(d.Reasons))
	for _, r := range d.Reasons {
		reasons = append(reasons, ReasonResponse{
			Code:        r.Code,
			Description: r.Description,
			Value:       r.Value,
			Threshold:   r.Threshold,
			EvidenceRef: r.EvidenceRef,
		})
	}

	resp := DecisionResponse{
		Outcome: d.Outcome.String(),
		Reasons: reasons,
		Policy: PolicyDTO{
			ID:      d.PolicyID,
			Version: d.PolicyVersion,
		},
	}

	if d.Outputs != nil {
		resp.Outputs = &DecisionOutputsDTO{
			CreditLimit:    d.Outputs.CreditLimit,
			ApprovedAmount: d.Outputs.ApprovedAmount,
		}
	}

	return resp
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: message,
		Code:  code,
	})
}
