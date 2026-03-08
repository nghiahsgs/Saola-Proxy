# Code Review Summary

## Scope
- Files reviewed: cmd/saola/main.go, internal/cli/{root,wrap,version,init-cmd,audit-cmd}.go, internal/scanner/{pattern,registry,scanner,patterns-builtin}.go, internal/sanitizer/{mapping-table,sanitizer,rehydrator}.go, internal/wrapper/{wrapper,io-bridge,signal-handler}.go, internal/config/{config,defaults}.go, internal/audit/{session,writer,logger}.go, Makefile, .goreleaser.yaml, .github/workflows/{ci,release}.yaml
- Lines of code analyzed: ~750 (all non-test source)
- Review focus: Full codebase, security + correctness + architecture

## Overall Assessment

Well-structured, idiomatic Go for its scope. Build is clean, all tests pass with race detector enabled. Core logic (scan → sanitize → rehydrate) is correct and thread-safe. Several real bugs and one security concern found.

---

## Critical Issues

None that cause data loss or remote code execution, but the item below is a security concern.

---

## High Priority Findings

### H1 — Security: Chunk-boundary secret splitting (io-bridge.go:8, wrapper/io-bridge.go)
**File:** `internal/wrapper/io-bridge.go:8` — `const chunkSize = 4096`

The IOBridge reads in 4096-byte chunks and applies `Sanitize` per chunk. A secret that straddles a chunk boundary (e.g. a JWT that starts in chunk N and ends in chunk N+1) will NOT be detected by the regex. The sanitizer never sees the full token — only a partial prefix and a partial suffix, neither of which matches.

This is the core security invariant of the tool and it can silently fail.

**Fix:** Buffer output until a newline (or until N bytes with no newline), then flush. Alternatively, maintain a small overlap window between chunks equal to `maxSecretLength` bytes, scanning `overlap + chunk` and emitting only the overlap-confirmed-clean prefix.

Simplest safe option for line-oriented output (logs, kubectl, cat):
```go
// use bufio.Scanner with ScanLines, then sanitize each complete line.
// For truly binary/non-line output, a sliding-window approach is needed.
scanner := bufio.NewScanner(b.src)
for scanner.Scan() {
    line := scanner.Text() + "\n"
    if _, err := io.WriteString(b.dst, b.process(line)); err != nil { ... }
}
```

---

### H2 — Bug: Dead / misleading code in scanner.go (scanner.go:41)
**File:** `internal/scanner/scanner.go:41`

```go
result := raw[:0]        // line 41 — assigns a zero-length slice backed by raw
result = make([]Match, 0, len(resolved))  // line 42 — immediately replaces it
```

Line 41 is dead code. The `raw[:0]` assignment is overwritten on line 42, so the comment about "reuse backing array" is false — `resolved` is the post-overlap-removal slice, `raw` is the pre-removal slice, and no backing array is actually reused. The only effect is that `raw` is kept alive until GC. This is not a correctness bug but is confusing and the comment is wrong.

**Fix:** Remove line 41 entirely:
```go
result := make([]Match, 0, len(resolved))
```

---

### H3 — Bug: Non-unique session IDs cause silent audit overwrite (audit/session.go:36, audit/writer.go:32)
**File:** `internal/audit/session.go:36`

```go
ID: now.Format("20060102-150405"),  // second-level granularity
```

If `saola wrap` is called twice within the same second, `session.ID` is identical for both. `WriteAudit` writes to `session-<ID>.json`, silently overwriting the first session's file.

**Fix:** Use a higher-resolution or random suffix:
```go
ID: fmt.Sprintf("%s-%d", now.Format("20060102-150405"), now.UnixNano()%1_000_000),
// or
ID: now.Format("20060102-150405.000"),  // millisecond precision
```

---

### H4 — Bug: `runPipe` goroutines may not finish before `cmd.Wait` returns (wrapper/wrapper.go:125-133)

In `runPipe`, three goroutines read from child stdout/stderr. `cmd.Wait()` is called on line 135. After `Wait`, the pipe file-descriptors are closed by the OS, but the goroutines writing to `os.Stdout` / `os.Stderr` may still be mid-write or in the `io.WriteString` call. The function then returns, and the output is truncated.

This is a classic race: the goroutine for stderr (line 133) and stdout (line 127) have no synchronisation with the `cmd.Wait` return path. The PTY path (`runPTY`) correctly runs `outBridge` in the foreground, but `runPipe` does not.

**Fix:** Use a `sync.WaitGroup`:
```go
var wg sync.WaitGroup
wg.Add(3)
go func() { defer wg.Done(); b := NewIOBridge(os.Stdin, stdinPipe, ...); _ = b.Run(ctx) }()
go func() { defer wg.Done(); b := NewIOBridge(stdoutPipe, os.Stdout, ...); _ = b.Run(ctx) }()
go func() { defer wg.Done(); b := NewIOBridge(stderrPipe, os.Stderr, ...); _ = b.Run(ctx) }()
cmd.Wait()
wg.Wait()
```

---

## Medium Priority Improvements

### M1 — `audit/logger.go`: `SetupLogger` is never called
`SetupLogger` is exported and fully implemented, but no call site exists in the codebase. `LogLevel` is parsed from config but the logger is never wired up. This is dead code / YAGNI violation.

**Fix:** Either call it from `wrap.go` after loading config, or remove it until needed.

---

### M2 — `audit/session.go`: `RecordRehydration` is never called
`RecordRehydration` exists and `TotalRehydrated` is tracked in the session summary, but no call site exists. The rehydrator never notifies the session.

**Fix:** Either add a callback on `Rehydrator` analogous to `Sanitizer.OnDetection`, or remove the field until the feature is built.

---

### M3 — `config/config.go:41`: `os.UserHomeDir()` error silently ignored
```go
home, _ := os.UserHomeDir()
return filepath.Join(home, ".saola")
```
If `UserHomeDir()` fails (containers without `/etc/passwd`, certain CI environments), `home` is `""` and the path becomes `"/.saola"` — a root-owned directory. Same pattern in `audit/writer.go:14`.

**Fix:**
```go
home, err := os.UserHomeDir()
if err != nil {
    home = os.TempDir() // or return error
}
```

---

### M4 — `patterns-builtin.go`: `rePhoneUS` is highly ambiguous (false-positive risk)
```go
rePhoneUS = regexp.MustCompile(`(?:\+?1[-.\s]?)?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`)
```
This matches any 10-digit sequence with optional separators. Common outputs like port numbers, timestamps, version strings with 3+3+4 digit groups, or numeric IDs will match. Example: `"2024-01-15 10:30:4512"` contains a false match. This will cause unexpected redaction of non-PII output.

**Fix:** Add stricter word-boundary anchors and require at least one separator:
```go
`(?:(?:\+?1[-.\s])?\(?[0-9]{3}\)?[-.\s][0-9]{3}[-.\s][0-9]{4})\b`
```
(Requiring separators eliminates pure digit stream false positives.)

---

### M5 — `patterns-builtin.go`: `reAWSAccessKey` has no word-boundary protection
```go
reAWSAccessKey = regexp.MustCompile(`AKIA[0-9A-Z]{16}`)
```
Will match `AKIA` anywhere in a longer string (e.g. a base64 payload that happens to contain `AKIA`). Low-risk but worth anchoring:
```go
`\bAKIA[0-9A-Z]{16}\b`
```

---

### M6 — `wrap.go:104-106`: `os.Exit` called from inside `RunE`

```go
os.Exit(exitCode)
return nil // unreachable
```

Calling `os.Exit` from within `cobra.Command.RunE` bypasses cobra's cleanup (including deferred functions registered by cobra itself). The comment acknowledges it's unreachable. This pattern prevents `defer` from running and makes the function untestable.

**Fix:** Return a sentinel error or use `cobra.Command.PostRun` for exit code propagation. A common pattern:
```go
// Store exitCode, return a typed error, handle in Execute().
```
Alternatively this is acceptable for a CLI that owns its own lifecycle, but the `// unreachable` comment is a code smell.

---

## Low Priority Suggestions

### L1 — `scanner.go:41`: Remove dead `result = raw[:0]` line (covered in H2).

### L2 — `.github/workflows/ci.yaml`: No linter step. Consider adding `golangci-lint` to catch issues earlier.

### L3 — `.goreleaser.yaml`: Missing Windows builds. Only `linux` and `darwin` listed in `goos`. If Windows support is planned, add it; if not, document explicitly.

### L4 — `audit/writer.go`: Audit directory is hardcoded to `~/.saola/audit` regardless of `XDG_CONFIG_HOME`. Inconsistent with `ConfigDir()` which respects XDG. Should use `filepath.Join(ConfigDir(), "audit")`.

### L5 — `wrap.go`: `DisableFlagParsing: true` on `wrapCmd` means `globalConfigPath` (set on root) is not applied when `--config` comes after `wrap`. Behavior may surprise users. Documented limitation acceptable, but worth a comment.

---

## Positive Observations

- Thread-safety is consistently handled: `MappingTable`, `Session` both use `sync.RWMutex` / `sync.Mutex` correctly.
- `resolveOverlaps` algorithm is correct: sort by start descending, keep longest on tie — proper greedy interval scheduling.
- Right-to-left replacement in `Sanitizer.Sanitize` correctly preserves byte offsets.
- Sensitive keys filtered from slog output — good defensive logging.
- Config file permissions: 0600 for config, 0700 for directories — correct.
- `placeholderRe` in rehydrator is tight: `[A-Z][A-Z0-9_]*_\d+` — low collision risk with AI output.
- PTY vs pipe mode detection (`term.IsTerminal`) is the correct approach.
- Pattern registry's `Disable` by name is O(n) linear scan — fine at 13 built-ins.

---

## Recommended Actions (Priority Order)

1. **H1 (Critical UX)** — Fix chunk-boundary secret splitting. Line-buffer `IOBridge` output, or use sliding overlap. This is the most impactful correctness issue.
2. **H4 (Bug)** — Add `sync.WaitGroup` to `runPipe` to prevent truncated output.
3. **H3 (Bug)** — Use millisecond-precision or random suffix for session IDs.
4. **M3** — Handle `os.UserHomeDir()` error in `ConfigDir()` and `AuditDir()`.
5. **M1 / M2** — Wire up `SetupLogger` and `RecordRehydration`, or remove them.
6. **M4** — Tighten `rePhoneUS` regex to reduce false positives.
7. **L4** — Make `AuditDir()` respect XDG config path.
8. **H2** — Remove dead `result = raw[:0]` line.

---

## Metrics

- Build: PASS (go build ./...)
- Vet: PASS (go vet ./...)
- Tests: PASS (go test -race -count=1 ./... — all packages)
- Race detector: PASS
- Test coverage (total): 69.3%
  - scanner: 98.6%
  - sanitizer: 97.5%
  - audit: 83.6%
  - config: 83.3%
  - wrapper: 44.8% (PTY path untested — expected for integration code)
  - cli: 0% (CLI integration — acceptable, needs e2e tests)
- Linting: No formal linter in CI

## Unresolved Questions

- Is Windows support planned? `.goreleaser.yaml` omits it but the tool is useful on Windows too.
- Is `SetupLogger` intended to replace `fmt.Fprintf(os.Stderr, ...)` calls? Currently disconnected from the runtime config.
- Should `saola audit` respect `--config` flag (for XDG dir)? Currently hardcoded to `~/.saola/audit`.
