package scanner

// PatternRegistry holds all registered PII detection patterns.
type PatternRegistry struct {
	patterns []Pattern
}

// NewRegistry creates an empty PatternRegistry.
func NewRegistry() *PatternRegistry {
	return &PatternRegistry{}
}

// Register adds a pattern to the registry.
func (r *PatternRegistry) Register(p Pattern) {
	r.patterns = append(r.patterns, p)
}

// GetEnabled returns all currently enabled patterns.
func (r *PatternRegistry) GetEnabled() []Pattern {
	enabled := make([]Pattern, 0, len(r.patterns))
	for _, p := range r.patterns {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// GetAll returns all registered patterns (both enabled and disabled).
func (r *PatternRegistry) GetAll() []Pattern {
	out := make([]Pattern, len(r.patterns))
	copy(out, r.patterns)
	return out
}

// Disable disables the pattern with the given name.
func (r *PatternRegistry) Disable(name string) {
	for i := range r.patterns {
		if r.patterns[i].Name == name {
			r.patterns[i].Enabled = false
			return
		}
	}
}

// Enable enables the pattern with the given name.
func (r *PatternRegistry) Enable(name string) {
	for i := range r.patterns {
		if r.patterns[i].Name == name {
			r.patterns[i].Enabled = true
			return
		}
	}
}
