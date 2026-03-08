# Phase 6: Audit Logging

**Status**: `[ ]` Not Started
**Depends on**: Phase 3 (sanitizer mapping table), Phase 5 (config)
**Estimated effort**: 0.5 day

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 5 - Config System](./phase-05-config-system.md)
- Next: [Phase 7 - Build & Release](./phase-07-build-release.md)

## Overview

Session-level audit logging using Go's `slog` package. Logs statistical summaries of PII detections (pattern counts, session duration) without ever recording actual PII values. Provides a `saola audit` command to view recent session reports.

## Key Insights

- **Stats only, never PII values** - critical security requirement
- Go 1.21+ `slog` is sufficient, no external logging library needed
- `slog.ReplaceAttr` can be used to ensure no PII leaks into logs even if accidentally passed
- Session audit file: `~/.saola/audit/session-YYYYMMDD-HHMMSS.json`
- Keep audit simple: start time, end time, per-pattern match counts, total sanitized, wrapped command

## Requirements

- [ ] Session audit struct with start/end time, pattern counts, command info
- [ ] `slog`-based logger with safe attribute filtering
- [ ] Write session summary to audit file on exit
- [ ] `saola audit` command to list/view recent sessions
- [ ] Configurable via `audit_enabled` in config
- [ ] Never log PII values - only counts and pattern names

## Architecture

```
internal/audit/
  session.go           # Session stats tracking
  logger.go            # slog setup with safe filtering
  writer.go            # Write audit file on session end
  audit_test.go        # Tests
```

### Session Audit Format (JSON)

```json
{
  "session_id": "20260308-143022",
  "command": "claude",
  "start_time": "2026-03-08T14:30:22Z",
  "end_time": "2026-03-08T15:45:10Z",
  "duration_seconds": 4488,
  "detections": {
    "email": 3,
    "aws-access-key": 1,
    "github-token": 2
  },
  "total_sanitized": 6,
  "total_rehydrated": 4
}
```

### slog Safety Filter

```go
func safeReplaceAttr(groups []string, a slog.Attr) slog.Attr {
    // Strip any attribute that might contain PII
    if a.Key == "value" || a.Key == "original" || a.Key == "pii" {
        return slog.Attr{} // drop it
    }
    return a
}
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/audit/session.go` | Session stats tracker |
| `internal/audit/logger.go` | Safe slog configuration |
| `internal/audit/writer.go` | Audit file writer |
| `internal/audit/audit_test.go` | Tests |
| `internal/cli/audit.go` | `saola audit` command |
| `internal/sanitizer/mapping-table.go` | Provides Stats() data |

## Implementation Steps

1. Create `session.go`:
   - `type Session struct` - ID, command, start/end time, detection counts map, totals
   - `NewSession(command string) *Session`
   - `RecordDetection(patternName string)` - increment counter (thread-safe)
   - `RecordRehydration()` - increment rehydration counter
   - `End()` - set end time, calculate duration
   - `Summary() SessionSummary` - exportable struct for JSON
2. Create `logger.go`:
   - `SetupLogger(level slog.Level) *slog.Logger`
   - Uses `slog.NewJSONHandler` with `safeReplaceAttr`
   - Logs to stderr (not stdout, to avoid mixing with wrapped process output)
3. Create `writer.go`:
   - `WriteAudit(session *Session) error`
   - Creates `~/.saola/audit/` directory if not exists
   - Writes `session-{ID}.json`
   - Uses `json.MarshalIndent` for readability
4. Create `internal/cli/audit.go`:
   - `saola audit` - list recent sessions (last 10)
   - `saola audit --session <ID>` - show specific session detail
   - Reads from `~/.saola/audit/` directory
   - Tabular output: date, command, duration, total detections
5. Integrate with wrapper:
   - Create session at wrapper start
   - Pass session to sanitizer for recording detections
   - Call `session.End()` + `WriteAudit()` in wrapper cleanup
6. Create `audit_test.go`:
   - Test session recording
   - Test concurrent RecordDetection
   - Test JSON output format
   - Test audit file creation
   - Test audit command output

## Todo List

- [ ] `session.go` - session stats tracking
- [ ] `logger.go` - safe slog setup
- [ ] `writer.go` - audit file writer
- [ ] `audit.go` - saola audit command
- [ ] Integrate session with wrapper lifecycle
- [ ] Integrate detection recording with sanitizer
- [ ] Unit tests
- [ ] Verify no PII appears in audit files

## Success Criteria

1. Session audit file created in `~/.saola/audit/` after each wrapped session
2. Audit JSON contains only counts and metadata, zero PII values
3. `saola audit` lists recent sessions with summary
4. Logger's safe filter drops any accidentally-passed PII attributes
5. Audit disabled when `audit_enabled: false` in config
6. Concurrent detection recording works without races

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| PII accidentally logged | High | slog ReplaceAttr filter, code review, test verification |
| Audit directory fills up over time | Low | Future: add retention policy (e.g., keep last 30 days) |
| Audit file write fails | Low | Log warning, don't crash wrapper |

## Security Considerations

- **CRITICAL**: Audit files must NEVER contain PII values, only pattern names and counts
- Audit files stored with 0600 permissions
- slog safety filter is defense-in-depth, not primary protection
- Code review checklist: search for `.Value` or `.original` in any slog call

## Next Steps

Proceed to [Phase 7 - Build & Release](./phase-07-build-release.md) for build automation, cross-compilation, and release process.
