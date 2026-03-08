# Saola Proxy - MVP Implementation Plan

> **Saola** (Vietnamese Unicorn) - A privacy gateway for AI coding assistants.
> Single Go binary that wraps CLI tools and strips PII from I/O streams.

## Summary

Saola Proxy intercepts communication between developers and AI coding assistants (Claude Code, Copilot CLI, etc.) by wrapping the target process in a PTY. It scans stdin/stdout for PII patterns (API keys, emails, tokens, etc.), replaces them with deterministic placeholders, and can rehydrate responses. MVP = CLI wrapper mode only.

## Tech Stack

- **Language**: Go 1.22+
- **CLI**: spf13/cobra
- **PTY**: creack/pty
- **Terminal**: golang.org/x/term
- **Config**: gopkg.in/yaml.v3
- **Logging**: stdlib slog

## Phases

| # | Phase | Status | Priority | Link |
|---|-------|--------|----------|------|
| 1 | Project Setup | `not-started` | P0 | [phase-01](./phase-01-project-setup.md) |
| 2 | PII Scanner Engine | `not-started` | P0 | [phase-02](./phase-02-pii-scanner-engine.md) |
| 3 | Sanitizer & Rehydrator | `not-started` | P0 | [phase-03](./phase-03-sanitizer-rehydrator.md) |
| 4 | CLI Wrapper | `not-started` | P0 | [phase-04](./phase-04-cli-wrapper.md) |
| 5 | Config System | `not-started` | P1 | [phase-05](./phase-05-config-system.md) |
| 6 | Audit & Logging | `not-started` | P1 | [phase-06](./phase-06-audit-logging.md) |
| 7 | Build & Release | `not-started` | P1 | [phase-07](./phase-07-build-release.md) |

## Architecture (High-Level)

```
User Terminal
    │
    ▼
┌──────────┐    stdin     ┌──────────┐    stdin     ┌──────────┐
│  saola   │ ──────────►  │ sanitizer│ ──────────►  │ wrapped  │
│  CLI     │              │ (strip)  │              │ process  │
│          │  ◄────────── │ (rehydr) │  ◄────────── │ (claude) │
└──────────┘    stdout    └──────────┘    stdout    └──────────┘
                               │
                          ┌────▼────┐
                          │ scanner │
                          │ (regex) │
                          └─────────┘
```

## Key Decisions

1. **Regex over ML** - KISS. Regex patterns are fast, auditable, zero-dependency
2. **PTY wrapping over HTTPS proxy** - MVP simplicity. HTTPS proxy = Phase 2 (future)
3. **Deterministic placeholders** - `[EMAIL_1]` not random UUIDs. Enables rehydration + readability
4. **Session-scoped mapping** - PII mappings live in memory per session, never persisted to disk

## Dependency Graph

```
Phase 1 (Setup) → Phase 2 (Scanner) → Phase 3 (Sanitizer) → Phase 4 (Wrapper)
                                                                    ↑
Phase 5 (Config) ───────────────────────────────────────────────────┘
Phase 6 (Audit) ────────────────────────────────────────────────────┘
Phase 7 (Build) ← all phases
```

## Success Criteria (MVP)

- [ ] `saola wrap -- claude` intercepts and sanitizes PII in both directions
- [ ] 13+ PII patterns detected (API keys, tokens, emails, SSN, credit cards, etc.)
- [ ] Deterministic placeholder replacement with rehydration
- [ ] YAML config for custom patterns and whitelists
- [ ] Session audit stats (no PII logged)
- [ ] Cross-platform binary (linux/darwin, amd64/arm64)
