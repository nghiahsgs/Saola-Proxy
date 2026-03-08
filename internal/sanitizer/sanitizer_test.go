package sanitizer_test

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// newTestPipeline returns a wired-up sanitizer + rehydrator sharing one table.
func newTestPipeline() (*sanitizer.Sanitizer, *sanitizer.Rehydrator, *sanitizer.MappingTable) {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	s := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	san := sanitizer.NewSanitizer(s, table)
	reh := sanitizer.NewRehydrator(table)
	return san, reh, table
}

// ---- Round-trip tests ----

func TestRoundTripEmail(t *testing.T) {
	san, reh, _ := newTestPipeline()
	original := "Please contact john@example.com for details."
	sanitized := san.Sanitize(original)
	if strings.Contains(sanitized, "john@example.com") {
		t.Error("sanitized text should not contain original email")
	}
	restored := reh.Rehydrate(sanitized)
	if restored != original {
		t.Errorf("round-trip failed:\n  want: %q\n   got: %q", original, restored)
	}
}

func TestRoundTripSSN(t *testing.T) {
	san, reh, _ := newTestPipeline()
	original := "SSN on file: 123-45-6789."
	restored := reh.Rehydrate(san.Sanitize(original))
	if restored != original {
		t.Errorf("round-trip SSN failed: got %q", restored)
	}
}

func TestRoundTripMultiplePII(t *testing.T) {
	san, reh, _ := newTestPipeline()
	original := "Email: alice@example.com, SSN: 987-65-4321, IP: 10.0.0.1"
	sanitized := san.Sanitize(original)
	restored := reh.Rehydrate(sanitized)
	if restored != original {
		t.Errorf("multi-PII round-trip failed:\n  want: %q\n   got: %q", original, restored)
	}
}

// ---- Determinism test ----

func TestDeterminism(t *testing.T) {
	san, _, table := newTestPipeline()
	text := "user@example.com"

	p1 := san.Sanitize(text)
	p2 := san.Sanitize(text)

	if p1 != p2 {
		t.Errorf("same PII should produce same placeholder: %q vs %q", p1, p2)
	}
	stats := table.Stats()
	// Should still be counter=1 despite two sanitize calls for the same value.
	if stats["EMAIL"] != 1 {
		t.Errorf("expected EMAIL counter=1, got %d", stats["EMAIL"])
	}
}

// ---- Counter increment test ----

func TestCounterIncrement(t *testing.T) {
	san, _, table := newTestPipeline()

	san.Sanitize("first@example.com")
	san.Sanitize("second@example.com")
	san.Sanitize("third@example.com")

	stats := table.Stats()
	if stats["EMAIL"] != 3 {
		t.Errorf("expected EMAIL counter=3, got %d", stats["EMAIL"])
	}
}

func TestPlaceholderNaming(t *testing.T) {
	san, reh, _ := newTestPipeline()
	text := "first@example.com and second@example.com"
	sanitized := san.Sanitize(text)

	if !strings.Contains(sanitized, "[EMAIL_1]") {
		t.Errorf("expected [EMAIL_1] in %q", sanitized)
	}
	if !strings.Contains(sanitized, "[EMAIL_2]") {
		t.Errorf("expected [EMAIL_2] in %q", sanitized)
	}
	restored := reh.Rehydrate(sanitized)
	if restored != text {
		t.Errorf("round-trip failed: got %q", restored)
	}
}

// ---- Empty / no-PII inputs ----

func TestEmptyText(t *testing.T) {
	san, reh, _ := newTestPipeline()
	if out := san.Sanitize(""); out != "" {
		t.Errorf("expected empty string, got %q", out)
	}
	if out := reh.Rehydrate(""); out != "" {
		t.Errorf("expected empty string, got %q", out)
	}
}

func TestNoPIIText(t *testing.T) {
	san, reh, _ := newTestPipeline()
	text := "Hello, world! No secrets here."
	if out := san.Sanitize(text); out != text {
		t.Errorf("no-PII text modified: got %q", out)
	}
	if out := reh.Rehydrate(text); out != text {
		t.Errorf("no-placeholder text modified: got %q", out)
	}
}

// ---- Brackets that aren't placeholders ----

func TestBracketsNotPlaceholders(t *testing.T) {
	_, reh, _ := newTestPipeline()
	text := "Error [404] and [not_a_placeholder] and [lower_1] remain unchanged."
	if out := reh.Rehydrate(text); out != text {
		t.Errorf("non-placeholder brackets changed: got %q", out)
	}
}

// ---- Unknown placeholder is left as-is ----

func TestUnknownPlaceholderPreserved(t *testing.T) {
	_, reh, _ := newTestPipeline()
	text := "Response mentions [EMAIL_99] which we never registered."
	if out := reh.Rehydrate(text); out != text {
		t.Errorf("unknown placeholder should not be replaced: got %q", out)
	}
}

// ---- Concurrent access test ----

func TestConcurrentAccess(t *testing.T) {
	san, reh, _ := newTestPipeline()
	var wg sync.WaitGroup
	const workers = 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			email := fmt.Sprintf("user%d@example.com", n)
			text := fmt.Sprintf("Email is %s and SSN is 123-45-6789", email)
			sanitized := san.Sanitize(text)
			restored := reh.Rehydrate(sanitized)
			if !strings.Contains(restored, email) {
				t.Errorf("goroutine %d: email not restored in %q", n, restored)
			}
		}(i)
	}
	wg.Wait()
}

// ---- Pattern name to placeholder category ----

func TestPatternNameConversion(t *testing.T) {
	// aws-access-key should become AWS_ACCESS_KEY in placeholder
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	s := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	san := sanitizer.NewSanitizer(s, table)

	text := "key: AKIAIOSFODNN7EXAMPLE"
	sanitized := san.Sanitize(text)
	if !strings.Contains(sanitized, "[AWS_ACCESS_KEY_1]") {
		t.Errorf("expected [AWS_ACCESS_KEY_1] in %q", sanitized)
	}
}
