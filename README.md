# Saola Proxy

> The Asian Unicorn — A privacy gateway for AI coding assistants.

Saola Proxy intercepts communication between you and AI coding tools (Claude Code, etc.), automatically detecting and masking sensitive data (API keys, emails, tokens, credentials) before it reaches external servers. When responses come back, Saola rehydrates the placeholders with your original values.

**Your secrets never leave your machine.**

## Features

- **13+ PII patterns** — Detects AWS keys, GitHub tokens, Stripe keys, emails, SSN, credit cards, JWTs, private keys, and more
- **Bidirectional sanitization** — Masks outgoing data, restores incoming responses
- **Deterministic placeholders** — `[EMAIL_1]`, `[AWS_ACCESS_KEY_1]` for readable, reversible masking
- **Configurable** — YAML config for custom patterns, whitelists, and settings
- **Audit logging** — Session stats without storing PII
- **Single binary** — Zero runtime dependencies, cross-platform
- **100% Local** — No cloud, no telemetry, open source

## Quick Start

### Install

```bash
# From source
go install github.com/nguyennghia/saola-proxy/cmd/saola@latest

# Or download binary from GitHub Releases
```

### Usage

```bash
# Wrap any CLI tool
saola wrap -- claude

# Wrap with explicit config
saola --config ~/.saola/config.yaml wrap -- claude

# Initialize default config
saola init

# View audit stats
saola audit
```

### Example

```
$ echo "My API key is AKIAIOSFODNN7EXAMPLE and email is john@company.com" | saola wrap -- cat
My API key is [AWS_ACCESS_KEY_1] and email is [EMAIL_1]
```

## How It Works

```
You ──► Saola (sanitize) ──► AI Tool ──► AI Server
You ◄── Saola (rehydrate) ◄── AI Tool ◄── AI Server
```

1. Saola wraps the target process in a pseudo-terminal
2. Outgoing text is scanned for PII patterns
3. Detected PII is replaced with deterministic placeholders
4. AI processes the sanitized text
5. Responses containing placeholders are restored to original values

## Built-in Patterns

| Pattern | Category | Example |
|---------|----------|---------|
| AWS Access Key | credential | `AKIA...` |
| GitHub Token | secret | `ghp_...` |
| Stripe Key | secret | `sk_live_...` |
| Generic API Key | secret | `api_key=...` |
| Private Key | credential | `-----BEGIN RSA PRIVATE KEY-----` |
| JWT | secret | `eyJ...` |
| Connection String | credential | `postgres://...` |
| Email | pii | `user@example.com` |
| SSN | pii | `123-45-6789` |
| Credit Card | pii | Visa, MC, Amex |
| US Phone | pii | `(555) 123-4567` |
| IPv4 Address | pii | `192.168.1.1` |
| Env Variable | secret | `PASSWORD=...` |

## Configuration

```bash
saola init  # Creates ~/.saola/config.yaml
```

```yaml
version: 1
log_level: info
audit_enabled: true

patterns:
  disabled:
    - phone-us
    - ipv4-address
  custom:
    - name: internal-token
      category: secret
      regex: "INTERNAL_[A-Z0-9]{32}"
      description: "Internal API tokens"

whitelist:
  - "127.0.0.1"
  - "localhost"
  - "example.com"
  - "test@example.com"
```

## Audit

```bash
saola audit  # View recent sessions
```

Audit logs track detection counts (not PII values):
```json
{
  "session_id": "20260308-143022",
  "command": "claude",
  "detections": {"email": 3, "aws-access-key": 1},
  "total_sanitized": 4
}
```

## Development

```bash
make build    # Build binary
make test     # Run tests with race detector
make coverage # Generate coverage report
make clean    # Clean build artifacts
```

## License

MIT — see [LICENSE](LICENSE)

## Why "Saola"?

The [Saola](https://en.wikipedia.org/wiki/Saola) is one of the world's rarest mammals, found only in the Annamite Mountains of Vietnam and Laos. Known as the "Asian Unicorn," it's famous for being incredibly elusive — much like how your sensitive data should be when passing through AI tools.
