// Package wrapper provides a PTY-based CLI wrapper that transparently
// intercepts stdin/stdout to sanitize PII before it reaches AI tools
// and rehydrates AI responses with original values.
package wrapper

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"golang.org/x/term"
)

// Wrapper executes a child command with PTY interception, applying
// sanitization to stdout/stderr and rehydration to stdin.
type Wrapper struct {
	command    string
	args       []string
	sanitizer  *sanitizer.Sanitizer
	rehydrator *sanitizer.Rehydrator
}

// NewWrapper creates a Wrapper for the given command and arguments.
func NewWrapper(command string, args []string, san *sanitizer.Sanitizer, reh *sanitizer.Rehydrator) *Wrapper {
	return &Wrapper{
		command:    command,
		args:       args,
		sanitizer:  san,
		rehydrator: reh,
	}
}

// Run executes the child process and returns its exit code.
// If stdin is a terminal, PTY mode is used for full interactive support.
// If stdin is piped, simple pipe mode is used instead.
func (w *Wrapper) Run() (int, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return w.runPTY()
	}
	return w.runPipe()
}

// runPTY starts the child in a pseudo-terminal, bridges I/O with
// sanitization/rehydration, and forwards signals.
func (w *Wrapper) runPTY() (int, error) {
	cmd := exec.Command(w.command, w.args...)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 1, fmt.Errorf("pty start: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	// Put the real terminal into raw mode so keystrokes flow unmodified.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return 1, fmt.Errorf("make raw: %w", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Forward SIGWINCH and other signals to child.
	go HandleSignals(ctx, cancel, ptmx, cmd)

	// stdin → PTY: sanitize user input before it reaches the AI tool.
	// Chunk-based: interactive typing is character-by-character.
	inBridge := NewIOBridge(os.Stdin, ptmx, w.sanitizer.Sanitize)
	go func() { _ = inBridge.Run(ctx) }()

	// PTY → stdout: rehydrate AI responses so user sees original values.
	// Line-buffered: prevents placeholders from being missed at chunk boundaries.
	outBridge := NewLineBufferedIOBridge(ptmx, os.Stdout, w.rehydrator.Rehydrate)
	// Run outBridge in foreground to detect EOF from child exit.
	_ = outBridge.Run(ctx)

	// Wait for the process and extract exit code.
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

// runPipe handles the non-TTY case (stdin is a pipe or file).
// It uses standard pipe I/O without PTY.
func (w *Wrapper) runPipe() (int, error) {
	cmd := exec.Command(w.command, w.args...)

	// Wire up stdin with rehydration.
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return 1, fmt.Errorf("stdin pipe: %w", err)
	}

	// Wire up stdout/stderr pipes for sanitization.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return 1, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("start: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// stdin → child: sanitize user input before it reaches the AI tool.
	go func() {
		defer stdinPipe.Close()
		b := NewIOBridge(os.Stdin, stdinPipe, w.sanitizer.Sanitize)
		_ = b.Run(ctx)
	}()

	// child stdout → our stdout: rehydrate so user sees original values.
	wg.Add(1)
	go func() {
		defer wg.Done()
		b := NewLineBufferedIOBridge(stdoutPipe, os.Stdout, w.rehydrator.Rehydrate)
		_ = b.Run(ctx)
	}()

	// child stderr → our stderr: rehydrate so user sees original values.
	wg.Add(1)
	go func() {
		defer wg.Done()
		b := NewLineBufferedIOBridge(stderrPipe, os.Stderr, w.rehydrator.Rehydrate)
		_ = b.Run(ctx)
	}()

	if err := cmd.Wait(); err != nil {
		wg.Wait() // ensure all output is captured before returning
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		// io.EOF from wait is normal when pipes close.
		if err == io.EOF {
			return 0, nil
		}
		return 1, err
	}
	wg.Wait() // ensure all output is captured before returning
	return 0, nil
}
