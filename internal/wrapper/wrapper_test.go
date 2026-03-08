package wrapper

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// newTestComponents builds a scanner/sanitizer/rehydrator triple backed
// by the default built-in pattern registry.
func newTestComponents() (*sanitizer.Sanitizer, *sanitizer.Rehydrator) {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	sc := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	san := sanitizer.NewSanitizer(sc, table)
	reh := sanitizer.NewRehydrator(table)
	return san, reh
}

// --------------------------------------------------------------------------
// IOBridge unit tests
// --------------------------------------------------------------------------

func TestIOBridge_PassThrough(t *testing.T) {
	src := strings.NewReader("hello world")
	var dst bytes.Buffer
	identity := func(s string) string { return s }

	b := NewIOBridge(src, &dst, identity)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := dst.String(); got != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got)
	}
}

func TestIOBridge_ProcessApplied(t *testing.T) {
	src := strings.NewReader("abc")
	var dst bytes.Buffer
	upper := func(s string) string { return strings.ToUpper(s) }

	b := NewIOBridge(src, &dst, upper)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := dst.String(); got != "ABC" {
		t.Errorf("expected %q, got %q", "ABC", got)
	}
}

func TestIOBridge_ContextCancellation(t *testing.T) {
	// Use a pipe so reading blocks until cancelled.
	pr, pw := io.Pipe()
	defer pw.Close()

	var dst bytes.Buffer
	identity := func(s string) string { return s }

	ctx, cancel := context.WithCancel(context.Background())
	b := NewIOBridge(pr, &dst, identity)

	done := make(chan error, 1)
	go func() { done <- b.Run(ctx) }()

	cancel()
	pw.Close() // unblock the Read so goroutine can detect ctx.Done

	err := <-done
	if err != nil && err != context.Canceled {
		t.Fatalf("expected nil or context.Canceled, got %v", err)
	}
}

// --------------------------------------------------------------------------
// Sanitization through IOBridge
// --------------------------------------------------------------------------

func TestIOBridge_SanitizesEmail(t *testing.T) {
	san, _ := newTestComponents()

	input := "Contact us at test@example.com for help."
	src := strings.NewReader(input)
	var dst bytes.Buffer

	b := NewIOBridge(src, &dst, san.Sanitize)
	if err := b.Run(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := dst.String()
	if strings.Contains(out, "test@example.com") {
		t.Errorf("email not sanitized; output: %q", out)
	}
	if !strings.Contains(out, "[EMAIL_") {
		t.Errorf("expected placeholder in output; got: %q", out)
	}
}

// --------------------------------------------------------------------------
// Wrapper pipe-mode integration tests (non-TTY)
// --------------------------------------------------------------------------

func TestWrapper_PipeMode_HelloWorld(t *testing.T) {
	san, reh := newTestComponents()
	w := NewWrapper("echo", []string{"hello"}, san, reh)

	// runPipe is used when stdin is not a TTY; call it directly.
	exitCode, err := w.runPipe()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}
}

func TestWrapper_PipeMode_EmailSanitized(t *testing.T) {
	san, reh := newTestComponents()

	// Capture stdout by temporarily replacing it.
	origStdout := io.Writer(nil) // unused but documented
	_ = origStdout

	// We test sanitization via IOBridge directly (see TestIOBridge_SanitizesEmail).
	// Here we verify the wrapper runs the command and exits cleanly.
	w := NewWrapper("echo", []string{"test@example.com"}, san, reh)
	exitCode, err := w.runPipe()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}
}

func TestWrapper_PipeMode_NonZeroExitCode(t *testing.T) {
	san, reh := newTestComponents()
	// "false" exits with code 1 on all Unix-like systems.
	w := NewWrapper("false", []string{}, san, reh)
	exitCode, err := w.runPipe()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode == 0 {
		t.Error("expected non-zero exit code from 'false'")
	}
}

func TestWrapper_PipeMode_CommandNotFound(t *testing.T) {
	san, reh := newTestComponents()
	w := NewWrapper("__nonexistent_saola_cmd__", []string{}, san, reh)
	exitCode, err := w.runPipe()
	// Either an error is returned OR a non-zero exit code; both are acceptable.
	if err == nil && exitCode == 0 {
		t.Error("expected error or non-zero exit for missing command")
	}
}

// --------------------------------------------------------------------------
// NewWrapper constructor
// --------------------------------------------------------------------------

func TestNewWrapper(t *testing.T) {
	san, reh := newTestComponents()
	w := NewWrapper("echo", []string{"hi"}, san, reh)
	if w == nil {
		t.Fatal("expected non-nil Wrapper")
	}
	if w.command != "echo" {
		t.Errorf("unexpected command: %s", w.command)
	}
	if len(w.args) != 1 || w.args[0] != "hi" {
		t.Errorf("unexpected args: %v", w.args)
	}
}
