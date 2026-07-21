package harness

// Registry looks up Harness implementations by Type.
type Registry struct {
	harnesses map[Type]Harness
}

// NewRegistry builds a Registry from the given harnesses, keyed by each
// harness's Type().
func NewRegistry(harnesses ...Harness) *Registry {
	m := make(map[Type]Harness, len(harnesses))
	for _, h := range harnesses {
		m[h.Type()] = h
	}
	return &Registry{harnesses: m}
}

// Get returns the harness registered for t, if any.
func (r *Registry) Get(t Type) (Harness, bool) {
	h, ok := r.harnesses[t]
	return h, ok
}

// Default returns the Claude harness, or nil if it is not registered.
func (r *Registry) Default() Harness {
	h, ok := r.Get(ClaudeType)
	if !ok {
		return nil
	}
	return h
}
