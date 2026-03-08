// Package scanner provides PII and secret detection via regex pattern matching.
package scanner

import "regexp"

// Pattern defines a PII detection rule.
type Pattern struct {
	Name        string
	Category    string // "secret", "pii", "credential"
	Regex       *regexp.Regexp
	Description string
	Enabled     bool
}

// Match represents a detected PII occurrence in text.
type Match struct {
	PatternName string
	Category    string
	Value       string // the actual secret/PII value (captured group if present)
	Start       int
	End         int
}
