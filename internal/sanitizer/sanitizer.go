package sanitizer

import (
	"sort"

	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// Sanitizer replaces detected PII/secrets with placeholder tokens.
type Sanitizer struct {
	scanner     *scanner.Scanner
	table       *MappingTable
	OnDetection func(patternName string) // optional callback invoked for each detection
}

// NewSanitizer creates a Sanitizer backed by the given scanner and mapping table.
func NewSanitizer(s *scanner.Scanner, t *MappingTable) *Sanitizer {
	return &Sanitizer{scanner: s, table: t}
}

// Sanitize scans text for PII and replaces each occurrence with a placeholder.
// Replacements are applied right-to-left to preserve byte offsets.
// If OnDetection is set, it is called once per match with the pattern name.
func (s *Sanitizer) Sanitize(text string) string {
	matches := s.scanner.Scan(text)
	if len(matches) == 0 {
		return text
	}

	// Sort descending by Start so right-to-left replacement keeps offsets valid.
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Start > matches[j].Start
	})

	result := []byte(text)
	for _, m := range matches {
		placeholder := s.table.GetOrCreate(m.Value, m.PatternName)
		// Replace the full matched span (Start..End) with the placeholder.
		result = append(result[:m.Start], append([]byte(placeholder), result[m.End:]...)...)
		if s.OnDetection != nil {
			s.OnDetection(m.PatternName)
		}
	}
	return string(result)
}
