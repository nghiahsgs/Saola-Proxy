# Phase 3: Sanitizer & Rehydrator

**Status**: `[ ]` Not Started
**Depends on**: Phase 2 (scanner)
**Estimated effort**: 1 day

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 2 - PII Scanner](./phase-02-pii-scanner-engine.md)
- Next: [Phase 4 - CLI Wrapper](./phase-04-cli-wrapper.md)

## Overview

Bidirectional replacement engine. **Sanitizer**: takes scanner matches and replaces PII with deterministic placeholders (`[EMAIL_1]`, `[API_KEY_2]`). **Rehydrator**: reverses placeholders back to original values. Uses a session-scoped mapping table for consistency (same PII always gets same placeholder).

## Key Insights

- **Two-pass algorithm** (from research): Pass 1 finds all matches + resolves conflicts. Pass 2 replaces in reverse offset order to preserve positions
- **Deterministic placeholders**: Same PII value always maps to same placeholder within a session. Enables consistent rehydration
- **Mapping table** is bidirectional: `originalValue <-> placeholder`
- **Thread-safe**: wrapper may process stdin/stdout concurrently, mapping needs mutex
- **Counter per category**: `[EMAIL_1]`, `[EMAIL_2]`, `[AWS_ACCESS_KEY_1]` - counter scoped per pattern name

## Requirements

- [ ] `MappingTable` - bidirectional map with mutex, counter per pattern category
- [ ] `Sanitizer.Sanitize(text string) string` - scan + replace PII with placeholders
- [ ] `Rehydrator.Rehydrate(text string) string` - reverse placeholders to original values
- [ ] Two-pass replacement algorithm (reverse offset order)
- [ ] Deterministic: same input PII → same placeholder across calls
- [ ] Thread-safe mapping operations
- [ ] Unit tests covering round-trip sanitize→rehydrate

## Architecture

```
internal/sanitizer/
  mapping-table.go     # Bidirectional PII ↔ placeholder map
  sanitizer.go         # Sanitize(text) - PII → placeholder
  rehydrator.go        # Rehydrate(text) - placeholder → PII
  sanitizer_test.go    # Round-trip and edge case tests
```

### Core Algorithm (Two-Pass)

```
Sanitize(text):
  1. matches = scanner.Scan(text)          // Phase 2
  2. Sort matches by Start ascending
  3. Resolve overlaps (longest match wins, drop contained matches)
  4. For each match (reverse order by Start):
     a. placeholder = mappingTable.GetOrCreate(match.Value, match.PatternName)
     b. text = text[:match.Start] + placeholder + text[match.End:]
  5. Return text

Rehydrate(text):
  1. Find all placeholder patterns [PATTERN_NAME_N] in text
  2. For each placeholder:
     a. original = mappingTable.GetOriginal(placeholder)
     b. Replace placeholder with original
  3. Return text
```

### MappingTable Structure

```go
type MappingTable struct {
    mu             sync.RWMutex
    toPlaceholder  map[string]string  // "real@email.com" → "[EMAIL_1]"
    toOriginal     map[string]string  // "[EMAIL_1]" → "real@email.com"
    counters       map[string]int     // "EMAIL" → 2
}
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/sanitizer/mapping-table.go` | Bidirectional map |
| `internal/sanitizer/sanitizer.go` | PII → placeholder |
| `internal/sanitizer/rehydrator.go` | Placeholder → PII |
| `internal/sanitizer/sanitizer_test.go` | Tests |
| `internal/scanner/scanner.go` | Dependency - provides matches |

## Implementation Steps

1. Create `mapping-table.go`:
   - `NewMappingTable() *MappingTable`
   - `GetOrCreate(value, patternName string) string` - returns existing or creates new placeholder
   - `GetOriginal(placeholder string) (string, bool)` - reverse lookup
   - `Stats() map[string]int` - returns counter snapshot (for audit, Phase 6)
   - All methods use `sync.RWMutex`
2. Create `sanitizer.go`:
   - `NewSanitizer(scanner *scanner.Scanner, table *MappingTable) *Sanitizer`
   - `Sanitize(text string) string` - two-pass algorithm
   - Placeholder format: `[PATTERN_NAME_N]` where N is counter (e.g., `[EMAIL_1]`)
3. Create `rehydrator.go`:
   - `NewRehydrator(table *MappingTable) *Rehydrator`
   - `Rehydrate(text string) string` - find `[A-Z_]+_\d+]` patterns, reverse lookup
4. Create `sanitizer_test.go`:
   - Test single PII type sanitize + rehydrate round-trip
   - Test multiple PII types in one string
   - Test determinism: same PII → same placeholder across calls
   - Test different PII values of same type get incremented counters
   - Test overlapping matches
   - Test empty string, no PII
   - Test text with bracket characters that aren't placeholders
   - Concurrent access test (goroutines calling Sanitize simultaneously)

## Todo List

- [ ] `mapping-table.go` - bidirectional map with mutex
- [ ] `sanitizer.go` - Sanitize() with two-pass algorithm
- [ ] `rehydrator.go` - Rehydrate() with reverse lookup
- [ ] Unit tests: round-trip
- [ ] Unit tests: determinism
- [ ] Unit tests: overlapping matches
- [ ] Unit tests: concurrency safety
- [ ] Unit tests: edge cases

## Success Criteria

1. `Rehydrate(Sanitize(text)) == text` for all inputs containing PII
2. Same PII value produces same placeholder across multiple Sanitize calls
3. Different PII values of same type get different counters (`[EMAIL_1]`, `[EMAIL_2]`)
4. No data races under concurrent access (`go test -race`)
5. Placeholder format is `[UPPER_SNAKE_CASE_N]` consistently
6. Text without PII passes through unchanged

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Placeholder collision with real text | Low | Bracket format unlikely in code; can add prefix like `[SAOLA_` if needed |
| Memory growth from mapping table | Low | Session-scoped, dies with process. Typical session has <100 unique PII values |
| Rehydration regex matches non-placeholder brackets | Medium | Use strict pattern: `\[(?:EMAIL|AWS_ACCESS_KEY|...)\d+\]` matching only known types |

## Security Considerations

- Mapping table holds PII in memory only, never written to disk
- Table cleared on process exit (no cleanup needed)
- Stats() method exposes counts only, not values - safe for audit logging
- Rehydrator only operates on stdout (AI responses), not on data sent to external services

## Next Steps

Proceed to [Phase 4 - CLI Wrapper](./phase-04-cli-wrapper.md) to integrate scanner+sanitizer into the PTY wrapping layer.
