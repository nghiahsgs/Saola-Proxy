# Phase 2: PII Scanner Engine

**Status**: `[ ]` Not Started
**Depends on**: Phase 1 (project structure)
**Estimated effort**: 1 day

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 1 - Project Setup](./phase-01-project-setup.md)
- Next: [Phase 3 - Sanitizer & Rehydrator](./phase-03-sanitizer-rehydrator.md)

## Overview

Build the regex-based PII detection engine. Defines a `Pattern` type, a `PatternRegistry` for managing detection rules, and a `Scanner` that finds all PII matches in a given text. This is the core detection layer - it finds but does not replace.

## Key Insights

- 13 validated regex patterns from research cover most common secrets/PII
- Patterns must be compiled once at startup (regexp.MustCompile), not per-scan
- Scanner returns match structs with type, offset, length - sanitizer handles replacement
- Registry pattern allows users to add custom patterns via config (Phase 5)
- Must handle overlapping matches: longest match wins

## Requirements

- [ ] Define `Pattern` struct: Name, Category, Regex, Description, Enabled
- [ ] Define `Match` struct: PatternName, Value, Start, End
- [ ] Implement `PatternRegistry` - register/list/enable/disable patterns
- [ ] Ship 13 built-in patterns (see list below)
- [ ] Implement `Scanner.Scan(text string) []Match` - returns all matches sorted by offset
- [ ] Handle overlapping matches (longest wins)
- [ ] Unit tests with >90% coverage on scanner

## Architecture

```
internal/scanner/
  pattern.go           # Pattern and Match types
  registry.go          # PatternRegistry - manages pattern collection
  scanner.go           # Scanner - uses registry to find matches
  patterns-builtin.go  # 13 default patterns
  scanner_test.go      # Comprehensive tests
```

### Type Definitions

```go
type Pattern struct {
    Name        string
    Category    string         // "secret", "pii", "credential"
    Regex       *regexp.Regexp
    Description string
    Enabled     bool
}

type Match struct {
    PatternName string
    Value       string
    Start       int
    End         int
}

type Scanner struct {
    registry *PatternRegistry
}
```

## Built-in Patterns (13)

| # | Name | Category | Regex Summary |
|---|------|----------|---------------|
| 1 | `aws-access-key` | secret | `AKIA[0-9A-Z]{16}` |
| 2 | `aws-secret-key` | secret | 40-char base64 after known prefixes |
| 3 | `github-token` | secret | `(ghp\|gho\|ghs\|ghr\|github_pat)_[A-Za-z0-9_]{36,}` |
| 4 | `stripe-key` | secret | `(sk\|pk)_(test\|live)_[A-Za-z0-9]{24,}` |
| 5 | `generic-api-key` | secret | Key-value pattern after `api[_-]?key` headers |
| 6 | `private-key` | secret | `-----BEGIN (RSA\|EC\|DSA\|OPENSSH) PRIVATE KEY-----` block |
| 7 | `jwt` | credential | `eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+` |
| 8 | `connection-string` | credential | DB connection URIs (postgres://, mysql://, mongodb://) |
| 9 | `email` | pii | Standard email regex |
| 10 | `ssn` | pii | `\b\d{3}-\d{2}-\d{4}\b` |
| 11 | `credit-card` | pii | Luhn-valid 13-19 digit patterns (Visa, MC, Amex) |
| 12 | `phone-us` | pii | US phone formats |
| 13 | `ip-address` | pii | IPv4 dotted notation (with validation) |

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/scanner/pattern.go` | Core types |
| `internal/scanner/registry.go` | Pattern management |
| `internal/scanner/scanner.go` | Scan logic |
| `internal/scanner/patterns-builtin.go` | Default rules |
| `internal/scanner/scanner_test.go` | Tests |

## Implementation Steps

1. Create `pattern.go` - define `Pattern`, `Match` types
2. Create `registry.go` - `PatternRegistry` with `Register()`, `List()`, `GetEnabled()` methods
3. Create `patterns-builtin.go` - function `DefaultPatterns() []Pattern` returning all 13
4. Create `scanner.go`:
   - `NewScanner(registry *PatternRegistry) *Scanner`
   - `Scan(text string) []Match` - iterate enabled patterns, collect all matches
   - `resolveOverlaps(matches []Match) []Match` - sort by start, longest wins on overlap
5. Create `scanner_test.go`:
   - Test each of the 13 patterns individually with positive/negative cases
   - Test overlap resolution
   - Test empty input, no matches
   - Test multiple matches of different types in one string
   - Benchmark test for scan performance

## Todo List

- [ ] `pattern.go` - Pattern and Match types
- [ ] `registry.go` - PatternRegistry
- [ ] `patterns-builtin.go` - 13 default patterns
- [ ] `scanner.go` - Scanner.Scan() with overlap resolution
- [ ] `scanner_test.go` - unit tests per pattern
- [ ] `scanner_test.go` - overlap resolution tests
- [ ] `scanner_test.go` - edge case tests (empty, unicode, multiline)
- [ ] `scanner_test.go` - benchmark

## Success Criteria

1. All 13 patterns detect their target PII correctly (true positives)
2. Patterns reject non-PII content (minimal false positives)
3. Overlapping matches resolved correctly (longest wins)
4. Scanner handles empty strings, unicode, multiline text
5. Benchmark: scanning 1KB text < 1ms
6. Test coverage > 90%

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Regex false positives on code | Medium | Whitelist config in Phase 5, conservative patterns |
| IP address matching too broad | Medium | Validate octets 0-255, skip common non-PII IPs (127.0.0.1, 0.0.0.0) |
| Credit card false positives on numbers | Medium | Luhn check validation |
| Performance on large outputs | Low | Pre-compiled regex, benchmark validates |

## Security Considerations

- Scanner only detects, never stores PII persistently
- Test fixtures must use synthetic PII, never real data
- Pattern regexes should be reviewed for ReDoS vulnerability (catastrophic backtracking)

## Next Steps

Proceed to [Phase 3 - Sanitizer & Rehydrator](./phase-03-sanitizer-rehydrator.md) to build the replacement engine that uses scanner results.
