package scanner_test

import (
	"strings"
	"testing"

	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

func newTestScanner() *scanner.Scanner {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	return scanner.NewScanner(reg)
}

// ---- Pattern-level positive/negative tests ----

func TestAWSAccessKey(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"valid", "key: AKIAIOSFODNN7EXAMPLE", true},
		{"too short", "AKIAIOSFODNN7EXAM", false},
		{"not akia", "BKIAIOSFODNN7EXAMPLE1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := s.Scan(tt.input)
			got := containsPattern(matches, "aws-access-key")
			if got != tt.found {
				t.Errorf("aws-access-key: found=%v, want=%v for %q", got, tt.found, tt.input)
			}
		})
	}
}

func TestGitHubToken(t *testing.T) {
	s := newTestScanner()
	validToken := "ghp_" + strings.Repeat("A", 36)
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"ghp prefix", validToken, true},
		{"github_pat prefix", "github_pat_" + strings.Repeat("B", 36), true},
		{"too short", "ghp_ABC123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := s.Scan(tt.input)
			got := containsPattern(matches, "github-token")
			if got != tt.found {
				t.Errorf("github-token: found=%v, want=%v", got, tt.found)
			}
		})
	}
}

func TestStripeKey(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"live secret key", "sk_live_" + strings.Repeat("x", 24), true},
		{"test publishable key", "pk_test_" + strings.Repeat("y", 24), true},
		{"invalid prefix", "rk_live_" + strings.Repeat("z", 24), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := s.Scan(tt.input)
			if containsPattern(matches, "stripe-key") != tt.found {
				t.Errorf("stripe-key: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestGenericAPIKey(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name       string
		input      string
		found      bool
		wantValue  string
	}{
		{"api_key assignment", `api_key = "abcdefghijklmnopqrst"`, true, "abcdefghijklmnopqrst"},
		{"access_token colon", `access_token: secret1234567890ABCDEF`, true, "secret1234567890ABCDEF"},
		{"too short value", `api_key = "short"`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := s.Scan(tt.input)
			m := findPattern(matches, "generic-api-key")
			if (m != nil) != tt.found {
				t.Errorf("generic-api-key: found=%v, want=%v", m != nil, tt.found)
			}
			if tt.found && m != nil && m.Value != tt.wantValue {
				t.Errorf("generic-api-key value=%q, want %q", m.Value, tt.wantValue)
			}
		})
	}
}

func TestPrivateKey(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"rsa private key", "-----BEGIN RSA PRIVATE KEY-----", true},
		{"ec private key", "-----BEGIN EC PRIVATE KEY-----", true},
		{"public key - not matched", "-----BEGIN RSA PUBLIC KEY-----", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "private-key") != tt.found {
				t.Errorf("private-key: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestJWT(t *testing.T) {
	s := newTestScanner()
	// Real-world-style JWT (not valid, just format)
	validJWT := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"valid jwt", validJWT, true},
		{"not jwt", "eyJshort.eyJshort.short", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "jwt") != tt.found {
				t.Errorf("jwt: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestConnectionString(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"postgres", "postgres://user:pass@localhost:5432/db", true},
		{"mongodb", "mongodb://admin:secret@cluster.example.com/mydb", true},
		{"redis", "redis://localhost:6379", true},
		{"http - not matched", "http://example.com/path", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "connection-string") != tt.found {
				t.Errorf("connection-string: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestEmail(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"simple email", "user@example.com", true},
		{"email in text", "contact us at support@company.org today", true},
		{"no at sign", "notanemail.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "email") != tt.found {
				t.Errorf("email: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestSSN(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"valid ssn", "SSN: 123-45-6789", true},
		{"wrong format", "123-456-789", false},
		{"no dashes", "123456789", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "ssn") != tt.found {
				t.Errorf("ssn: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestCreditCard(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"visa 16 digit", "4111111111111111", true},
		{"mastercard", "5500005555555559", true},
		{"amex", "378282246310005", true},
		{"too short", "41111111111", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "credit-card") != tt.found {
				t.Errorf("credit-card: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestPhoneUS(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"dashes", "555-123-4567", true},
		{"with country code", "+1 555 123 4567", true},
		{"dots", "555.123.4567", true},
		{"too short", "555-123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "phone-us") != tt.found {
				t.Errorf("phone-us: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestIPv4(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name  string
		input string
		found bool
	}{
		{"valid ip", "192.168.1.100", true},
		{"loopback", "127.0.0.1", true},
		{"out of range", "256.0.0.1", false},
		{"partial", "192.168.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if containsPattern(s.Scan(tt.input), "ipv4-address") != tt.found {
				t.Errorf("ipv4-address: want found=%v for %q", tt.found, tt.input)
			}
		})
	}
}

func TestEnvVariable(t *testing.T) {
	s := newTestScanner()
	tests := []struct {
		name      string
		input     string
		found     bool
		wantValue string
	}{
		{"PASSWORD eq", `PASSWORD=supersecretpwd`, true, "supersecretpwd"},
		{"TOKEN colon", `TOKEN: mytoken12345678`, true, "mytoken12345678"},
		{"too short value", `SECRET=short`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := s.Scan(tt.input)
			m := findPattern(matches, "env-variable")
			if (m != nil) != tt.found {
				t.Errorf("env-variable: found=%v, want=%v", m != nil, tt.found)
			}
			if tt.found && m != nil && m.Value != tt.wantValue {
				t.Errorf("env-variable value=%q, want %q", m.Value, tt.wantValue)
			}
		})
	}
}

// ---- Scanner-level tests ----

func TestEmptyInput(t *testing.T) {
	s := newTestScanner()
	matches := s.Scan("")
	if len(matches) != 0 {
		t.Errorf("expected no matches for empty input, got %d", len(matches))
	}
}

func TestMultiplePIIInOneString(t *testing.T) {
	s := newTestScanner()
	text := "Contact john@example.com or call 555-123-4567 about SSN 123-45-6789"
	matches := s.Scan(text)
	if len(matches) < 3 {
		t.Errorf("expected at least 3 matches, got %d", len(matches))
	}
	if !containsPattern(matches, "email") {
		t.Error("expected email match")
	}
	if !containsPattern(matches, "ssn") {
		t.Error("expected ssn match")
	}
}

func TestWhitelistFiltering(t *testing.T) {
	s := newTestScanner()
	s.SetWhitelist([]string{"john@example.com"})
	matches := s.Scan("email: john@example.com and other@example.com")
	for _, m := range matches {
		if m.Value == "john@example.com" {
			t.Error("whitelisted value should not appear in matches")
		}
	}
	if !containsPattern(matches, "email") {
		t.Error("non-whitelisted email should still be matched")
	}
}

func TestOverlapResolution(t *testing.T) {
	// A connection string contains an email-like segment; the longer match should win.
	s := newTestScanner()
	text := "postgres://user@host:5432/db"
	matches := s.Scan(text)

	// Should not produce overlapping results.
	for i := 1; i < len(matches); i++ {
		if matches[i].Start < matches[i-1].End {
			t.Errorf("overlapping matches at positions [%d,%d) and [%d,%d)",
				matches[i-1].Start, matches[i-1].End, matches[i].Start, matches[i].End)
		}
	}
}

func TestSortedByStart(t *testing.T) {
	s := newTestScanner()
	text := "SSN 123-45-6789 and email user@example.com"
	matches := s.Scan(text)
	for i := 1; i < len(matches); i++ {
		if matches[i].Start < matches[i-1].Start {
			t.Errorf("matches not sorted by Start: %d before %d", matches[i].Start, matches[i-1].Start)
		}
	}
}

func TestDisablePattern(t *testing.T) {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	reg.Disable("email")
	s := scanner.NewScanner(reg)

	matches := s.Scan("user@example.com")
	if containsPattern(matches, "email") {
		t.Error("disabled pattern should not produce matches")
	}
}

func TestEnablePattern(t *testing.T) {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	reg.Disable("email")
	reg.Enable("email")
	s := scanner.NewScanner(reg)

	matches := s.Scan("user@example.com")
	if !containsPattern(matches, "email") {
		t.Error("re-enabled pattern should produce matches")
	}
}

// ---- Helpers ----

func containsPattern(matches []scanner.Match, name string) bool {
	return findPattern(matches, name) != nil
}

func findPattern(matches []scanner.Match, name string) *scanner.Match {
	for i := range matches {
		if matches[i].PatternName == name {
			return &matches[i]
		}
	}
	return nil
}
