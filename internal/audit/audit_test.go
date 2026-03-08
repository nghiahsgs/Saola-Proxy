package audit

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// --- Session tests ---

func TestNewSession(t *testing.T) {
	s := NewSession("cat /var/log/app.log")
	if s == nil {
		t.Fatal("NewSession returned nil")
	}
	if s.ID == "" {
		t.Error("expected non-empty ID")
	}
	if s.Command != "cat /var/log/app.log" {
		t.Errorf("unexpected command: %s", s.Command)
	}
	if s.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if s.Detections == nil {
		t.Error("Detections map should be initialised")
	}
}

func TestRecordDetection(t *testing.T) {
	s := NewSession("test")
	s.RecordDetection("email")
	s.RecordDetection("email")
	s.RecordDetection("jwt")

	if s.Detections["email"] != 2 {
		t.Errorf("expected 2 email detections, got %d", s.Detections["email"])
	}
	if s.Detections["jwt"] != 1 {
		t.Errorf("expected 1 jwt detection, got %d", s.Detections["jwt"])
	}
	if s.TotalSanitized != 3 {
		t.Errorf("expected TotalSanitized=3, got %d", s.TotalSanitized)
	}
}

func TestRecordDetection_Concurrent(t *testing.T) {
	s := NewSession("concurrent-test")
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.RecordDetection("email")
		}()
	}
	wg.Wait()

	if s.Detections["email"] != goroutines {
		t.Errorf("expected %d, got %d", goroutines, s.Detections["email"])
	}
	if s.TotalSanitized != goroutines {
		t.Errorf("expected TotalSanitized=%d, got %d", goroutines, s.TotalSanitized)
	}
}

func TestSummary(t *testing.T) {
	s := NewSession("kubectl logs")
	s.RecordDetection("aws-access-key")
	s.RecordRehydration()
	s.End()

	sum := s.Summary()

	if sum.ID != s.ID {
		t.Errorf("ID mismatch: %s vs %s", sum.ID, s.ID)
	}
	if sum.Command != "kubectl logs" {
		t.Errorf("unexpected command: %s", sum.Command)
	}
	if sum.Detections["aws-access-key"] != 1 {
		t.Error("expected 1 aws-access-key detection")
	}
	if sum.TotalRehydrated != 1 {
		t.Error("expected TotalRehydrated=1")
	}
	if sum.DurationMS < 0 {
		t.Error("DurationMS should be >= 0")
	}

	// Must be JSON-serialisable.
	data, err := json.Marshal(sum)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}

// --- WriteAudit / ListSessions tests ---

func overrideAuditDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// Monkey-patch home so AuditDir() points to tmp.
	// We do this by writing files directly to the path returned by AuditDir
	// after re-pointing home via env.
	t.Setenv("HOME", tmp)
	return filepath.Join(tmp, ".saola", "audit")
}

func TestWriteAudit(t *testing.T) {
	dir := overrideAuditDir(t)

	s := NewSession("test-write")
	s.RecordDetection("email")
	s.End()

	if err := WriteAudit(s); err != nil {
		t.Fatalf("WriteAudit error: %v", err)
	}

	expected := filepath.Join(dir, "session-"+s.ID+".json")
	info, err := os.Stat(expected)
	if err != nil {
		t.Fatalf("audit file not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600, got %o", info.Mode().Perm())
	}
}

func TestListSessions(t *testing.T) {
	overrideAuditDir(t)

	// Write two sessions with a small time gap.
	s1 := NewSession("first")
	s1.End()
	if err := WriteAudit(s1); err != nil {
		t.Fatal(err)
	}

	time.Sleep(1100 * time.Millisecond) // ensure different second in ID

	s2 := NewSession("second")
	s2.End()
	if err := WriteAudit(s2); err != nil {
		t.Fatal(err)
	}

	sessions, err := ListSessions(0)
	if err != nil {
		t.Fatalf("ListSessions error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Newest first.
	if sessions[0].Command != "second" {
		t.Errorf("expected 'second' first, got %s", sessions[0].Command)
	}
}

func TestListSessions_Limit(t *testing.T) {
	overrideAuditDir(t)

	for i := 0; i < 3; i++ {
		s := NewSession("cmd")
		s.End()
		if err := WriteAudit(s); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1100 * time.Millisecond)
	}

	sessions, err := ListSessions(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 (limit), got %d", len(sessions))
	}
}

// --- Logger tests ---

func TestSetupLogger_DropsSecretAttrs(t *testing.T) {
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if sensitiveKeys[a.Key] {
				return slog.Attr{}
			}
			return a
		},
	}
	logger := slog.New(slog.NewJSONHandler(&buf, opts))

	logger.Info("test event",
		slog.String("value", "super-secret"),
		slog.String("token", "abc123"),
		slog.String("safe_key", "harmless"),
	)

	output := buf.String()
	if bytes.Contains([]byte(output), []byte("super-secret")) {
		t.Error("logger leaked 'value' attribute")
	}
	if bytes.Contains([]byte(output), []byte("abc123")) {
		t.Error("logger leaked 'token' attribute")
	}
	if !bytes.Contains([]byte(output), []byte("harmless")) {
		t.Error("expected safe_key to be present in log output")
	}
}

func TestSetupLogger_Levels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}
	for _, lvl := range levels {
		logger := SetupLogger(lvl)
		if logger == nil {
			t.Errorf("SetupLogger(%q) returned nil", lvl)
		}
	}
}
