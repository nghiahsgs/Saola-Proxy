# Phase 4: CLI Wrapper (PTY)

**Status**: `[ ]` Not Started
**Depends on**: Phase 2 (scanner), Phase 3 (sanitizer)
**Estimated effort**: 1.5 days

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 3 - Sanitizer & Rehydrator](./phase-03-sanitizer-rehydrator.md)
- Next: [Phase 5 - Config System](./phase-05-config-system.md)

## Overview

PTY-based process wrapper. Spawns the target CLI (e.g., `claude`) in a pseudo-terminal, intercepts stdin and stdout streams, applies sanitizer on outbound data (user→AI) and rehydrator on inbound data (AI→user). Handles terminal raw mode, window resize signals, and graceful shutdown.

## Key Insights

- `creack/pty` handles PTY creation, start, and resize
- Must set terminal to raw mode (`golang.org/x/term`) so keystrokes pass through immediately
- **Bidirectional I/O**: Two goroutines - one for stdin→pty (sanitize), one for pty→stdout (rehydrate)
- Signal forwarding: SIGWINCH (window resize) must propagate to child PTY
- SIGINT/SIGTERM: forward to child, wait for exit, then clean up
- **Stream buffering challenge**: PII might span chunk boundaries. Use line-based buffering (flush on newline) for MVP. SSE-aware parsing deferred
- Must restore terminal state on exit (defer)

## Requirements

- [ ] Spawn child process with PTY via `creack/pty`
- [ ] Set terminal to raw mode, restore on exit
- [ ] Forward stdin to child PTY (with sanitization)
- [ ] Forward child PTY output to stdout (with rehydration)
- [ ] Handle SIGWINCH - resize child PTY
- [ ] Handle SIGINT/SIGTERM - forward to child, graceful shutdown
- [ ] Line-buffered sanitization to handle chunk boundaries
- [ ] Exit with child's exit code
- [ ] Wire into `wrap` cobra subcommand

## Architecture

```
internal/wrapper/
  wrapper.go           # Core PTY wrapper logic
  io-bridge.go         # Bidirectional I/O with sanitization
  signal-handler.go    # Signal forwarding (SIGWINCH, SIGINT, SIGTERM)
  wrapper_test.go      # Integration tests
```

### Flow Diagram

```
Terminal (raw mode)
    │
    ├── stdin ──► io-bridge ──► Sanitizer.Sanitize() ──► child PTY stdin
    │
    └── stdout ◄── io-bridge ◄── Rehydrator.Rehydrate() ◄── child PTY stdout
                                                              │
                                                         child process
                                                         (e.g., claude)
```

### Line Buffer Strategy

```
For each read chunk from source:
  1. Append to internal buffer
  2. While buffer contains '\n':
     a. Extract line (up to and including '\n')
     b. Process line through sanitizer/rehydrator
     c. Write processed line to destination
  3. If buffer exceeds max size (4KB), flush as-is (safety valve)
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/wrapper/wrapper.go` | PTY lifecycle management |
| `internal/wrapper/io-bridge.go` | Bidirectional I/O + sanitization |
| `internal/wrapper/signal-handler.go` | OS signal handling |
| `internal/cli/wrap.go` | Cobra subcommand wiring |
| `internal/sanitizer/sanitizer.go` | Outbound sanitization |
| `internal/sanitizer/rehydrator.go` | Inbound rehydration |

## Implementation Steps

1. Create `wrapper.go`:
   - `type Wrapper struct` - holds pty file, child cmd, sanitizer, rehydrator
   - `NewWrapper(command string, args []string, sanitizer, rehydrator) *Wrapper`
   - `Run() error`:
     a. Create exec.Command
     b. Start with `pty.Start(cmd)`
     c. Set terminal raw mode, defer restore
     d. Start signal handler
     e. Start I/O bridge goroutines
     f. Wait for child exit
     g. Return child exit code via `exec.ExitError`

2. Create `io-bridge.go`:
   - `type IOBridge struct` - source reader, dest writer, processor func
   - `NewIOBridge(src io.Reader, dst io.Writer, process func(string) string) *IOBridge`
   - `Run(ctx context.Context) error` - read loop with line buffering
   - Line buffer: accumulate until newline, then process and write
   - Safety valve: flush if buffer > 4KB without newline

3. Create `signal-handler.go`:
   - `HandleSignals(ctx context.Context, ptmx *os.File, cmd *exec.Cmd)`
   - Listen for SIGWINCH: `pty.InheritSize(os.Stdin, ptmx)`
   - Listen for SIGINT, SIGTERM: `cmd.Process.Signal(sig)`
   - Initial size sync on start

4. Update `internal/cli/wrap.go`:
   - Parse command and args from cobra args
   - Create scanner, mapping table, sanitizer, rehydrator
   - Create and run wrapper
   - `os.Exit()` with child's exit code

## Todo List

- [ ] `wrapper.go` - PTY lifecycle
- [ ] `io-bridge.go` - bidirectional I/O with line buffering
- [ ] `signal-handler.go` - SIGWINCH, SIGINT, SIGTERM
- [ ] Update `wrap.go` cobra command to wire everything
- [ ] Manual test: `saola wrap bash` - verify interactive shell works
- [ ] Manual test: `saola wrap echo "test@email.com"` - verify sanitization
- [ ] Integration test: spawn simple echo process, verify I/O passes through
- [ ] Integration test: verify PII in output gets sanitized

## Success Criteria

1. `saola wrap bash` opens interactive shell, all keystrokes work
2. PII typed into wrapped process gets sanitized before reaching child
3. PII in child's output gets rehydrated (placeholders replaced back)
4. Terminal resize (SIGWINCH) propagates correctly
5. Ctrl+C forwards to child, clean exit
6. Exit code matches child's exit code
7. Terminal state restored after exit (no broken terminal)

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Terminal escape sequences corrupted by sanitization | High | Line buffer helps; don't sanitize ANSI escape sequences. Can add escape sequence detection later |
| Chunk boundary splits PII | Medium | Line buffering handles most cases. 4KB safety valve prevents hang |
| Raw mode not restored on panic | High | Use defer + recover; also handle in signal handler |
| Non-TTY stdin (piped input) | Medium | Detect with `term.IsTerminal()`, skip raw mode if not TTY |

## Security Considerations

- Child process inherits limited env vars (configurable in Phase 5)
- PTY data never written to disk
- Sanitization happens before data leaves the process boundary
- If sanitizer panics, raw data must NOT be forwarded - fail closed

## Next Steps

Proceed to [Phase 5 - Config System](./phase-05-config-system.md) to add user-configurable patterns, whitelists, and settings.
