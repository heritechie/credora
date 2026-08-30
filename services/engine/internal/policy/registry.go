// Package policy implements the policy evaluation engine and policy registry.
//
// The registry holds executable policy definitions in Go code. This avoids
// serializing Go function closures to the database while keeping policy
// definitions deterministic and version-controlled.
//
// PolicyRepository implementations persist policy metadata in the database
// and combine it with executable definitions from the registry.
package policy

import (
	"fmt"
	"sync"

	"credora/internal/domain"
)

// Registry holds executable policy definitions in memory.
// It is the source of truth for policy logic (knockouts, rules, score thresholds).
// Policy metadata (name, description) may be persisted separately in the database.
type Registry struct {
	mu       sync.RWMutex
	policies map[string]domain.Policy // key: "policyID:version"
}

// NewRegistry creates a new empty policy registry.
func NewRegistry() *Registry {
	return &Registry{
		policies: make(map[string]domain.Policy),
	}
}

// Register adds a policy to the registry.
// It returns an error if a policy with the same ID and version is already registered.
func (r *Registry) Register(pol domain.Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := policyKey(pol.ID, pol.Version)
	if _, exists := r.policies[key]; exists {
		return fmt.Errorf("policy %s v%d already registered", pol.ID, pol.Version)
	}

	r.policies[key] = pol
	return nil
}

// Get retrieves a policy by ID and version.
// It returns the policy and true if found, or a zero value and false if not found.
func (r *Registry) Get(id string, version int) (domain.Policy, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := policyKey(id, version)
	pol, ok := r.policies[key]
	return pol, ok
}

// Has reports whether a policy with the given ID and version is registered.
func (r *Registry) Has(id string, version int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.policies[policyKey(id, version)]
	return ok
}

func policyKey(id string, version int) string {
	return fmt.Sprintf("%s:%d", id, version)
}
