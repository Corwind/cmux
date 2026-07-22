package harness

// Registry tracks Harness implementations by Type, along with their
// registration order and per-type section names.
type Registry struct {
	order   []Type
	entries map[Type]Harness
	names   map[Type]string
}

// NewRegistry returns an empty Registry. Harnesses are added via Register.
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[Type]Harness),
		names:   make(map[Type]string),
	}
}

// Register adds h to the registry under sectionName. The first harness ever
// registered becomes the Default.
func (r *Registry) Register(h Harness, sectionName string) {
	t := h.Type()
	if _, exists := r.entries[t]; !exists {
		r.order = append(r.order, t)
	}
	r.entries[t] = h
	r.names[t] = sectionName
}

// Get returns the harness registered for t, if any.
func (r *Registry) Get(t Type) (Harness, bool) {
	h, ok := r.entries[t]
	return h, ok
}

// SectionName returns the section name registered for t, falling back to
// string(t) when unset or empty.
func (r *Registry) SectionName(t Type) string {
	if name, ok := r.names[t]; ok && name != "" {
		return name
	}
	return string(t)
}

// Default returns the first harness registered, or nil if none have been
// registered.
func (r *Registry) Default() Harness {
	if len(r.order) == 0 {
		return nil
	}
	return r.entries[r.order[0]]
}

// All returns the registered Types in registration order.
func (r *Registry) All() []Type {
	return r.order
}
