package credstore

import (
	"fmt"
	"sort"
)

// Registry maps store kinds to their implementations.
type Registry struct {
	stores map[string]CredentialStore
}

func NewRegistry(stores map[string]CredentialStore) *Registry {
	return &Registry{stores: stores}
}

func (r *Registry) Store(kind string) (CredentialStore, error) {
	s, ok := r.stores[kind]
	if !ok {
		return nil, fmt.Errorf("unknown credential store %q (have: %v)", kind, r.Kinds())
	}
	return s, nil
}

func (r *Registry) Kinds() []string {
	kinds := make([]string, 0, len(r.stores))
	for k := range r.stores {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	return kinds
}
