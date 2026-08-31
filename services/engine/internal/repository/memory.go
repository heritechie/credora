package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"credora/internal/domain"
)

// MemoryRepository is an in-memory implementation of AssessmentRepository.
// Suitable for development, testing, and situations where PostgreSQL
// is not available. Not suitable for production use.
type MemoryRepository struct {
	mu   sync.RWMutex
	byID map[string]domain.Assessment
}

// NewMemoryRepository creates a new in-memory assessment repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byID: make(map[string]domain.Assessment),
	}
}

func (r *MemoryRepository) Create(_ context.Context, assessment domain.Assessment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[assessment.ID]; exists {
		return fmt.Errorf("assessment %s already exists", assessment.ID)
	}

	r.byID[assessment.ID] = assessment
	return nil
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (domain.Assessment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	a, exists := r.byID[id]
	if !exists {
		return domain.Assessment{}, fmt.Errorf("assessment %s not found", id)
	}
	return a, nil
}

func (r *MemoryRepository) Update(_ context.Context, assessment domain.Assessment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byID[assessment.ID]; !exists {
		return fmt.Errorf("assessment %s not found", assessment.ID)
	}

	r.byID[assessment.ID] = assessment
	return nil
}

func (r *MemoryRepository) List(_ context.Context, limit, offset int) ([]domain.Assessment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var assessments []domain.Assessment
	keys := make([]string, 0, len(r.byID))
	for k := range r.byID {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	start := 0
	if offset > 0 {
		start = offset
	}
	end := len(keys)
	if limit > 0 && limit < end-start {
		end = start + limit
	}

	for i := start; i < end; i++ {
		a := r.byID[keys[i]]
		assessments = append(assessments, a)
	}

	return assessments, nil
}
