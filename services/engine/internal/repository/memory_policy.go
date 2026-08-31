package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// MemoryPolicyRepository is an in-memory implementation of PolicyRepository.
// Suitable for development and testing.
type MemoryPolicyRepository struct {
	mu   sync.RWMutex
	byID map[string]PolicyMetadata // key: "id:version"
}

// NewMemoryPolicyRepository creates a new in-memory policy repository.
func NewMemoryPolicyRepository() *MemoryPolicyRepository {
	return &MemoryPolicyRepository{
		byID: make(map[string]PolicyMetadata),
	}
}

func (r *MemoryPolicyRepository) Get(_ context.Context, id string, version int) (PolicyMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := policyMetaKey(id, version)
	meta, ok := r.byID[key]
	if !ok {
		return PolicyMetadata{}, fmt.Errorf("policy %s v%d not found", id, version)
	}
	return meta, nil
}

func (r *MemoryPolicyRepository) Save(_ context.Context, meta PolicyMetadata) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := policyMetaKey(meta.ID, meta.Version)
	r.byID[key] = meta
	return nil
}

func (r *MemoryPolicyRepository) Exists(_ context.Context, id string, version int) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := policyMetaKey(id, version)
	_, ok := r.byID[key]
	return ok, nil
}

func (r *MemoryPolicyRepository) List(_ context.Context) ([]PolicyMetadata, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var metas []PolicyMetadata
	for _, meta := range r.byID {
		metas = append(metas, meta)
	}
	sort.Slice(metas, func(i, j int) bool {
		if metas[i].ID != metas[j].ID {
			return metas[i].ID < metas[j].ID
		}
		return metas[i].Version < metas[j].Version
	})

	return metas, nil
}

func policyMetaKey(id string, version int) string {
	return fmt.Sprintf("%s:%d", id, version)
}
