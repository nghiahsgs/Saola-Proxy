// Package sanitizer replaces PII/secrets with reversible placeholders
// and rehydrates them back from AI responses.
package sanitizer

import (
	"fmt"
	"strings"
	"sync"
)

// MappingTable provides thread-safe bidirectional mapping between
// PII values and their placeholder tokens.
type MappingTable struct {
	mu            sync.RWMutex
	toPlaceholder map[string]string // "real@email.com" → "[EMAIL_1]"
	toOriginal    map[string]string // "[EMAIL_1]" → "real@email.com"
	counters      map[string]int    // "EMAIL" → 2
}

// NewMappingTable creates an empty MappingTable.
func NewMappingTable() *MappingTable {
	return &MappingTable{
		toPlaceholder: make(map[string]string),
		toOriginal:    make(map[string]string),
		counters:      make(map[string]int),
	}
}

// GetOrCreate returns the placeholder for value, creating one if needed.
// patternName is converted to UPPER_SNAKE_CASE (hyphens → underscores).
func (m *MappingTable) GetOrCreate(value, patternName string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.toPlaceholder[value]; ok {
		return p
	}

	category := toUpperSnake(patternName)
	m.counters[category]++
	placeholder := fmt.Sprintf("[%s_%d]", category, m.counters[category])

	m.toPlaceholder[value] = placeholder
	m.toOriginal[placeholder] = value
	return placeholder
}

// GetOriginal returns the original value for a placeholder, if known.
func (m *MappingTable) GetOriginal(placeholder string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.toOriginal[placeholder]
	return v, ok
}

// Stats returns a copy of the per-category replacement counts.
func (m *MappingTable) Stats() map[string]int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]int, len(m.counters))
	for k, v := range m.counters {
		out[k] = v
	}
	return out
}

// toUpperSnake converts a pattern name like "aws-access-key" → "AWS_ACCESS_KEY".
func toUpperSnake(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}
