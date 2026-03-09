package scanner

import "sort"

// Scanner applies registered patterns to text and returns detected matches.
type Scanner struct {
	registry  *PatternRegistry
	whitelist map[string]bool
}

// NewScanner creates a Scanner using the given registry.
func NewScanner(registry *PatternRegistry) *Scanner {
	return &Scanner{
		registry:  registry,
		whitelist: make(map[string]bool),
	}
}

// SetWhitelist configures values that should never be flagged as PII.
func (s *Scanner) SetWhitelist(values []string) {
	s.whitelist = make(map[string]bool, len(values))
	for _, v := range values {
		s.whitelist[v] = true
	}
}

// Scan detects all PII/secret matches in text, resolving overlaps.
func (s *Scanner) Scan(text string) []Match {
	if text == "" {
		return nil
	}

	var raw []Match
	for _, p := range s.registry.GetEnabled() {
		raw = append(raw, matchesForPattern(p, text)...)
	}

	resolved := resolveOverlaps(raw)

	// Filter whitelisted values and sort by start position.
	result := make([]Match, 0, len(resolved))
	for _, m := range resolved {
		if !s.whitelist[m.Value] {
			result = append(result, m)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Start < result[j].Start
	})
	return result
}

// matchesForPattern returns all matches for a single pattern.
// For patterns with a capture group, Value is the captured group content.
func matchesForPattern(p Pattern, text string) []Match {
	// Check if pattern has subgroup by inspecting SubexpNames.
	hasCaptureGroup := len(p.Regex.SubexpNames()) > 1

	var matches []Match
	locs := p.Regex.FindAllStringSubmatchIndex(text, -1)
	for _, loc := range locs {
		fullStart, fullEnd := loc[0], loc[1]
		value := text[fullStart:fullEnd]

		if hasCaptureGroup {
			// Find the first non-empty capture group.
			for g := 1; g < len(loc)/2; g++ {
				gs, ge := loc[g*2], loc[g*2+1]
				if gs >= 0 && ge >= 0 {
					value = text[gs:ge]
					break
				}
			}
		}

		// Run optional post-match validation (e.g. Luhn for credit cards).
		if p.Validate != nil && !p.Validate(value) {
			continue
		}

		matches = append(matches, Match{
			PatternName: p.Name,
			Category:    p.Category,
			Value:       value,
			Start:       fullStart,
			End:         fullEnd,
		})
	}
	return matches
}

// resolveOverlaps removes overlapping matches, keeping the longer one.
func resolveOverlaps(matches []Match) []Match {
	if len(matches) == 0 {
		return nil
	}

	// Sort by start position; for ties prefer longer match.
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Start != matches[j].Start {
			return matches[i].Start < matches[j].Start
		}
		return (matches[i].End - matches[i].Start) > (matches[j].End - matches[j].Start)
	})

	result := []Match{matches[0]}
	for _, m := range matches[1:] {
		last := &result[len(result)-1]
		if m.Start < last.End {
			// Overlap: keep the longer one.
			if (m.End - m.Start) > (last.End - last.Start) {
				*last = m
			}
			// else discard m
		} else {
			result = append(result, m)
		}
	}
	return result
}
