# Research: Go PII Proxy Patterns for Saola

## 1. Go CLI Wrapping Patterns

### PTY & Interactive Terminal Handling
**Recommended Library**: [creack/pty](https://github.com/creack/pty) - Most popular Go PTY package.

**Core Pattern**:
- Use `pty.Start(cmd)` to launch process with pseudo-terminal
- Set stdin to raw mode via `term.MakeRaw()` before execution
- Listen for `SIGWINCH` signal to handle terminal resize events
- Forward resize via `pty.InheritSize()` to keep PTY dimensions synced
- Bidirectional I/O: `io.Copy()` for stdin→pty and pty→stdout

**Signal Forwarding**:
- Subscribe to `syscall.SIGWINCH` using `signal.Notify()`
- Forward other signals (SIGINT, SIGTERM) via `cmd.Process.Signal()`
- Restore terminal state on exit via `term.Restore()`

**Alternative Options**:
- [go-pty](https://pkg.go.dev/github.com/wideeyedreven/go-pty) - Cross-platform (Unix/Windows ConPty)
- [interactive package](https://pkg.go.dev/github.com/integrii/interactive) - Channel-based I/O wrapper

**Trade-off**: Direct PTY is lower-level but more control; interactive package simplifies for non-interactive use cases.

---

## 2. PII Detection Regex Patterns

Recommended patterns (validated by security research communities):

| **Type** | **Pattern** | **Notes** |
|----------|-----------|----------|
| AWS Access Key | `AKIA[0-9A-Z]{16}` | IAM keys only |
| GitHub Token | `ghp_[0-9a-zA-Z]{36}` | Personal access tokens |
| GitHub OAuth | `gho_[0-9a-zA-Z]{36}` | OAuth tokens |
| Stripe Secret | `sk_live_[0-9a-zA-Z]{24}` | Production key prefix |
| Stripe Public | `pk_live_[0-9a-zA-Z]{24}` | Public key prefix |
| Generic API Key | `api[_-]?key[=:\s]+['\"]?[a-zA-Z0-9_-]{20,}` | Broad catch-all |
| Email | `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}` | RFC 5322 simplified |
| US Phone | `(\+1)?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}` | With optional +1 |
| SSN | `\b\d{3}-\d{2}-\d{4}\b` | US format only |
| Credit Card (Luhn) | `\b[0-9]{13,19}\b` + Luhn validation | Must validate checksum |
| Private Key | `-----BEGIN [A-Z]+ PRIVATE KEY-----` | PEM format headers |
| JWT Token | `eyJ[A-Za-z0-9_-]+\.eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+` | Header.payload.signature |
| JDBC/DB String | `jdbc:[a-z]+://[a-zA-Z0-9._-]+[^[\s]` | Connection strings |

**Implementation approach**: Compile patterns once at init time. For production, use negative lookahead to avoid false positives (e.g., exclude test@test.com, test SSNs).

---

## 3. Existing Go PII Detection Libraries

### Primary Library: [mangle](https://github.com/grugnog/mangle)
- **Dedicated Go masking library** - Clean API for data sanitization
- Deterministic word replacement using corpus + user secret
- Suitable for string masking but limited pattern detection

### Secondary Options:
- **[Microsoft Presidio](https://github.com/microsoft/presidio)** - Industry standard PII framework
  - Supports NLP + pattern matching + customizable pipelines
  - Python-based; Go integration via gRPC/REST API
  - Detects: CC, SSN, names, locations, financial data, crypto wallets, phone numbers
  - Trade-off: External dependency, higher latency for CLI wrapper use case

- **[gonymizer](https://github.com/smithoss/gonymizer)** - PostgreSQL-focused (not suitable for CLI proxy)

**Recommendation**: Use regex patterns + simple replacement for MVP (YAGNI), integrate Presidio via HTTP service for v2 if needed.

---

## 4. Go MITM Proxy Libraries

### Primary: [elazarl/goproxy](https://github.com/elazarl/goproxy)
**Features**:
- HTTP & HTTPS (CONNECT tunneling) support
- HTTPS MITM with on-the-fly certificate generation
- Request/response manipulation hooks
- TLS certificate caching to avoid regeneration
- Handler-based architecture compatible with `net/http`

**Key Methods**:
```
OnRequest(matcher).Do(handler)
OnResponse(matcher).Do(handler)
HandleConnect(goproxy.AlwaysMitm) // For HTTPS eavesdropping
```

### Alternatives:
- **[go-mitmproxy](https://pkg.go.dev/github.com/lqqyt2423/go-mitmproxy)** - Full mitmproxy reimplementation in Go
- **[ScrapeOps/go-proxy-mitm](https://github.com/ScrapeOps/go-proxy-mitm)** - Lightweight HTTP proxy library

**Trade-off**: goproxy is mature & focused on HTTP/HTTPS; go-mitmproxy is more feature-complete but heavier.

---

## 5. Recommended Architecture for MVP

**Phase 1** (CLI Wrapper):
- Use `creack/pty` + `os/exec` for interactive CLI wrapping
- Implement regex-based PII detection on stdin/stdout lines
- Simple string replacement masking (*** or UUID-based placeholders)

**Phase 2** (HTTPS Proxy):
- Add `goproxy` MITM mode for API request/response interception
- Extend regex patterns to JSON/request body parsing

**Unresolved Questions**:
1. Should Saola preserve original values via local mapping (e.g., map original email → xxxxxx@example.com) for context preservation?
2. Will CLI wrapper need to handle binary data or only text streams?
3. Should HTTPS proxy mode require user cert installation or use transparent proxying?
4. Performance constraints for regex matching on large payloads?

---

## Sources

**PTY & CLI Wrapping**:
- [creack/pty GitHub](https://github.com/creack/pty)
- [os/exec Package Docs](https://pkg.go.dev/os/exec)
- [PTY Setup Tutorial](https://linuxvox.com/blog/write-program-that-pretends-to-be-a-tty/)

**PII Libraries**:
- [mangle - Go Data Masking](https://github.com/grugnog/mangle)
- [Microsoft Presidio](https://github.com/microsoft/presidio)

**Regex Patterns**:
- [secret-regex-list](https://github.com/h33tlit/secret-regex-list)
- [keyhunter - API Key Scanner](https://github.com/fadidevv/keyhunter)

**MITM Proxy**:
- [elazarl/goproxy](https://github.com/elazarl/goproxy)
- [go-mitmproxy](https://github.com/lqqyt2423/go-mitmproxy)
