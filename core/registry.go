package core

import "sort"

// Registry maps operation names to their implementations.
type Registry struct {
	ops map[string]Operation
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{ops: make(map[string]Operation)}
}

// Default is the process-wide registry. Operation packages register into it
// from their init() functions.
var Default = NewRegistry()

// Register adds an operation to the registry, keyed by its Meta().Name.
func (r *Registry) Register(op Operation) {
	r.ops[op.Meta().Name] = op
}

// Get looks up an operation by name.
func (r *Registry) Get(name string) (Operation, bool) {
	op, ok := r.ops[name]
	return op, ok
}

// All returns every registered operation, sorted by name for stable output.
func (r *Registry) All() []Operation {
	out := make([]Operation, 0, len(r.ops))
	for _, op := range r.ops {
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Meta().Name < out[j].Meta().Name
	})
	return out
}

// Register adds an operation to the default registry.
func Register(op Operation) { Default.Register(op) }
