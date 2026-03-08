package sanitizer

import (
	"regexp"
)

// placeholderRe matches tokens like [EMAIL_1], [AWS_ACCESS_KEY_2], etc.
var placeholderRe = regexp.MustCompile(`\[([A-Z][A-Z0-9_]*_\d+)\]`)

// Rehydrator restores original PII values in AI responses.
type Rehydrator struct {
	table         *MappingTable
	OnRehydration func() // called each time a placeholder is successfully replaced
}

// NewRehydrator creates a Rehydrator backed by the given mapping table.
func NewRehydrator(t *MappingTable) *Rehydrator {
	return &Rehydrator{table: t}
}

// Rehydrate replaces all known placeholders in text with their original values.
func (r *Rehydrator) Rehydrate(text string) string {
	return placeholderRe.ReplaceAllStringFunc(text, func(token string) string {
		if original, ok := r.table.GetOriginal(token); ok {
			if r.OnRehydration != nil {
				r.OnRehydration()
			}
			return original
		}
		return token // unknown placeholder, leave as-is
	})
}
