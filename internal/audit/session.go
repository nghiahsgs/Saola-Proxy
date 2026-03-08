// Package audit provides session tracking and structured logging for
// Saola Proxy sanitization runs.
package audit

import (
	"sync"
	"time"
)

// Session records statistics about a single saola wrap invocation.
type Session struct {
	mu              sync.Mutex
	ID              string
	Command         string
	StartTime       time.Time
	EndTime         time.Time
	Detections      map[string]int // pattern name → occurrence count
	TotalSanitized  int
	TotalRehydrated int
}

// SessionSummary is the JSON-serialisable view of a Session.
type SessionSummary struct {
	ID              string         `json:"id"`
	Command         string         `json:"command"`
	StartTime       time.Time      `json:"start_time"`
	EndTime         time.Time      `json:"end_time"`
	DurationMS      int64          `json:"duration_ms"`
	Detections      map[string]int `json:"detections"`
	TotalSanitized  int            `json:"total_sanitized"`
	TotalRehydrated int            `json:"total_rehydrated"`
}

// NewSession creates a Session with ID derived from the current timestamp.
func NewSession(command string) *Session {
	now := time.Now()
	return &Session{
		ID:         now.Format("20060102-150405-000"),
		Command:    command,
		StartTime:  now,
		Detections: make(map[string]int),
	}
}

// RecordDetection increments the occurrence count for patternName (thread-safe).
func (s *Session) RecordDetection(patternName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Detections[patternName]++
	s.TotalSanitized++
}

// RecordRehydration increments TotalRehydrated (thread-safe).
func (s *Session) RecordRehydration() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRehydrated++
}

// End marks the session as finished.
func (s *Session) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.EndTime = time.Now()
}

// Summary returns a snapshot of the session suitable for JSON marshalling.
func (s *Session) Summary() SessionSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	detCopy := make(map[string]int, len(s.Detections))
	for k, v := range s.Detections {
		detCopy[k] = v
	}

	var durMS int64
	if !s.EndTime.IsZero() {
		durMS = s.EndTime.Sub(s.StartTime).Milliseconds()
	}

	return SessionSummary{
		ID:              s.ID,
		Command:         s.Command,
		StartTime:       s.StartTime,
		EndTime:         s.EndTime,
		DurationMS:      durMS,
		Detections:      detCopy,
		TotalSanitized:  s.TotalSanitized,
		TotalRehydrated: s.TotalRehydrated,
	}
}
