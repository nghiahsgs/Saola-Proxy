# Saola Proxy Research: CLI Architecture & Audit Patterns
**Date:** 2026-03-08 | **Status:** Complete

---

## 1. Claude Code CLI Architecture

### Communication Protocol & Proxy Support
- **Protocol:** Respects standard proxy env vars (`HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`)
- **Streaming:** Uses Server-Sent Events (SSE) for real-time response streaming
- **Critical Issue:** Some proxies break SSE streams, causing hangs (e.g., "0 Tokens" issue on macOS)
- **Implication for Saola:** Must properly parse/forward SSE streams without buffering/filtering that breaks event boundaries

### Key Findings
- Claude Code is process-wrappable (confirmed via third-party proxy implementations)
- SSE stream integrity is fragile—any proxy must maintain stream boundaries
- Proxy errors manifest as hanging requests, not clear error messages

### Design Recommendation
Saola should:
- Act as transparent HTTP/HTTPS MITM proxy with SSE stream awareness
- Parse SSE format correctly (newline-delimited events)
- Forward streams without breaking event structure
- Sanitize request bodies, stream responses, rehydrate before returning to Claude Code

---

## 2. Sanitization/Rehydration Patterns

### Go Libraries Available
- **Mangle** (GitHub: grugnog/mangle): Deterministic word corpus replacement, preserves length/case/punctuation
- **PII** (GitHub: ln80/pii): Struct-level PII tagging with replace options
- **Go-Sanitize** (GitHub: mrz1836/go-sanitize): String normalization utilities

### Best Practices Identified
1. **Deterministic Placeholders:** Use sequential PLACEHOLDER{i} format—identical PII values map to same placeholder
2. **Overlapping Matches:** Mangle approach—process longest matches first to avoid partial replacements
3. **Multi-line PII:** Regex with `(?s)` flag to match across newlines (e.g., private keys)
4. **Text Position Preservation:** Map original offsets to placeholder offsets for accurate rehydration

### Critical Implementation Detail
- Search results don't specify how libraries handle overlapping patterns
- **Recommendation:** Implement custom two-pass algorithm:
  - Pass 1: Find all matches, resolve conflicts (longest wins)
  - Pass 2: Replace in reverse offset order to preserve positions

---

## 3. YAML Configuration in Go

### Best Practice Stack
- **Parser:** `gopkg.in/yaml.v3` (pure Go, canonical implementation)
- **Config Discovery:** Manual implementation needed for XDG Base Directory spec
- **Path Resolution Pattern:**
  ```
  ~/.saola/config.yaml (user override)
  $XDG_CONFIG_HOME/saola/config.yaml (if set)
  ~/.config/saola/config.yaml (XDG fallback)
  ```

### Key Recommendation
- gopkg.in/yaml.v3 handles parsing; separate logic required for XDG directory detection
- Use environment variables: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `HOME` for path construction

---

## 4. Audit Logging Patterns

### Go's slog Framework (Go 1.21+)
- **Structured Key-Value Logging:** Built-in `log/slog` package
- **Redaction Support:** `ReplaceAttr()` handlers can mask sensitive keys (passwords, tokens, API keys)
- **Compliance:** Enables audit trails without leaking PII

### DLP-Aware Logging Strategy
1. **Session Stats Only:** Log sanitization counts, patterns matched, NOT original PII
2. **Audit Record Format:**
   ```
   {
     "timestamp": "2026-03-08T...",
     "session_id": "hash",
     "patterns_sanitized": {"api_key": 3, "email": 2, "credit_card": 0},
     "rules_triggered": ["gpt_api_key", "email_pattern"],
     "request_size_bytes": 1024,
     "status": "ok"
   }
   ```
3. **Use slog.RemoveAttr/ReplaceAttr** to scrub any field matching sensitive keys

---

## 5. Similar Open-Source Projects

### Go Ecosystem Analysis
- **No direct matches** for Go-based AI privacy proxies (market gap!)
- Closest alternatives:
  - **Presidio** (Python): MSoft PII detection, not for Go
  - **llm-guard** (Python): LLM-specific filtering
  - **Kong AI Sanitizer** (Lua/Go plugin): Enterprise gateway plugin, not standalone

### Python Dominance
- Most DLP/PII tools are Python (Presidio, llm-guard, Instructor examples)
- Go ecosystem lacks mature privacy-focused proxy frameworks

### Opportunity
Saola fills a genuine gap: **lightweight, deterministic, Go-native privacy gateway for AI tooling**

---

## Summary Table

| Topic | Finding | Action |
|-------|---------|--------|
| **CLI Architecture** | SSE streaming protocol; must preserve stream boundaries | Implement SSE-aware MITM proxy |
| **Sanitization** | Mangle (deterministic), PII (struct tags) available; overlapping handling needs custom logic | Build two-pass offset-based replacement |
| **YAML Config** | gopkg.in/yaml.v3 standard; XDG support requires custom logic | Implement path discovery helper |
| **Audit Logging** | slog + ReplaceAttr for redaction; session stats only pattern | Use slog with custom handlers |
| **Competitors** | None in Go; Python dominates; market gap confirmed | Positioning: Go's lightweight alternative |

---

## Unresolved Questions
1. **Exact SSE stream format** in Claude Code responses—need to verify if events are chunked or single-line
2. **Performance impact** of two-pass sanitization on large responses (streaming vs. buffering tradeoff)
3. **XDG directory precedence** across macOS, Linux, Windows WSL—implementation variance
4. **Rehydration accuracy** for nested JSON/YAML—how to map placeholders back through structured formats?

---

## Sources
- [GitHub - fuergaosi233/claude-code-proxy](https://github.com/fuergaosi233/claude-code-proxy)
- [GitHub - nielspeter/claude-code-proxy](https://github.com/nielspeter/claude-code-proxy)
- [Fixing Claude Code "0 Tokens" Hang: macOS Proxy and SSE Streaming](https://shiqimei.github.io/posts/claude-code-zero-tokens-hang.html)
- [gopkg.in/yaml.v3 - Go Packages](https://pkg.go.dev/gopkg.in/yaml.v3)
- [GitHub - grugnog/mangle: Sanitization/data masking library for Go](https://github.com/grugnog/mangle)
- [GitHub - ln80/pii: Go library to protect Personal Data](https://github.com/ln80/pii)
- [GitHub - mrz1836/go-sanitize](https://github.com/mrz1836/go-sanitize)
- [Structured Logging with slog - The Go Programming Language](https://go.dev/blog/slog)
- [slog package - log/slog - Go Packages](https://pkg.go.dev/log/slog)
- [Audit Logging for Data Loss Prevention](https://hoop.dev/blog/audit-logging-for-data-loss-prevention-ensuring-data-security)
- [AI PII Sanitization - Kong Docs](https://developer.konghq.com/plugins/ai-sanitizer/)
- [PII Sanitization for Agentic AI - Kong Inc.](https://konghq.com/blog/enterprise/building-pii-sanitization-for-llms-and-agentic-ai)
