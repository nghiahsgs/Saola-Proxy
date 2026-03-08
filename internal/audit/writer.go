package audit

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AuditDir returns the directory where audit files are stored (~/.saola/audit/).
// Returns an error if the home directory cannot be determined.
func AuditDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".saola", "audit"), nil
}

// WriteAudit serialises session to a JSON file in AuditDir().
// File is written with 0600 permissions.
func WriteAudit(session *Session) error {
	dir, err := AuditDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	summary := session.Summary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "session-"+session.ID+".json")
	return os.WriteFile(path, data, 0600)
}

// ListSessions reads all session-*.json files from AuditDir(), sorts them by
// StartTime descending, and returns up to limit entries.
// A limit of 0 returns all entries.
func ListSessions(limit int) ([]SessionSummary, error) {
	dir, err := AuditDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []SessionSummary
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "session-") || !strings.HasSuffix(name, ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue // skip unreadable files
		}

		var s SessionSummary
		if err := json.Unmarshal(data, &s); err != nil {
			continue // skip corrupt files
		}
		sessions = append(sessions, s)
	}

	// Sort newest first.
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartTime.After(sessions[j].StartTime)
	})

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}
