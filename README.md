# Saola Proxy

> The Asian Unicorn — A privacy gateway for AI coding assistants.

Saola Proxy intercepts API calls between you and AI coding tools (Claude Code, etc.), automatically detecting and masking sensitive data (API keys, emails, tokens, credentials) before it reaches external servers.

**Your secrets never leave your machine.**

## Features

- **HTTPS Proxy mode** — Intercepts API calls at network level, works with any AI tool
- **13+ PII patterns** — Detects AWS keys, GitHub tokens, Stripe keys, emails, SSN, credit cards, JWTs, private keys, and more
- **Deterministic placeholders** — `[EMAIL_1]`, `[AWS_ACCESS_KEY_1]` for readable masking
- **Configurable** — YAML config for custom patterns, whitelists, and settings
- **Audit logging** — Session stats without storing PII
- **Single binary** — Zero runtime dependencies, cross-platform
- **100% Local** — No cloud, no telemetry, open source

## Quick Start

### 1. Install

```bash
# Option 1: Build from source (requires Go 1.22+)
git clone https://github.com/nghiahsgs/Saola-Proxy.git
cd Saola-Proxy
make build
sudo cp bin/saola /usr/local/bin/

# Option 2: go install
go install github.com/nguyennghia/saola-proxy/cmd/saola@latest
```

### 2. Setup CA Certificate (one-time)

Saola needs a trusted CA certificate to intercept HTTPS traffic:

```bash
saola setup-ca
# Enter sudo password when prompted
```

This generates a local CA at `~/.saola/ca.crt` and installs it to your system trust store.

### 3. Start the Proxy

```bash
# Terminal 1: Start Saola proxy
saola proxy
```

```bash
# Terminal 2: Run Claude Code through the proxy
NODE_EXTRA_CA_CERTS=~/.saola/ca.crt HTTPS_PROXY=http://localhost:8080 claude
```

### 4. Create an Alias (recommended)

Add to your `~/.zshrc` or `~/.bashrc`:

```bash
alias claude-safe='NODE_EXTRA_CA_CERTS=~/.saola/ca.crt HTTPS_PROXY=http://localhost:8080 claude'
```

Then just type `claude-safe` — Saola sanitizes all API calls transparently.

## How It Works

```
You type: "Fix the bug, my email is john@company.com"
                          ↓
Saola intercepts the API call to api.anthropic.com
                          ↓
Anthropic receives: "Fix the bug, my email is [EMAIL_1]"
                          ↓
AI responds with [EMAIL_1] (never sees real email)
                          ↓
You see the response (with [EMAIL_1] placeholder)
```

Saola acts as an HTTPS MITM proxy — it intercepts traffic **only** to `api.anthropic.com`, sanitizes PII in request bodies, and passes everything else through untouched.

## Commands

```bash
saola setup-ca           # Generate and install CA certificate (one-time)
saola proxy              # Start HTTPS proxy on :8080
saola proxy --port 9090  # Start on custom port
saola init               # Create default config at ~/.saola/config.yaml
saola audit              # View sanitization stats from past sessions
saola version            # Print version
```

## Built-in Patterns

| Pattern | Category | Example |
|---------|----------|---------|
| AWS Access Key | secret | `AKIA...` |
| GitHub Token | secret | `ghp_...` |
| Stripe Key | secret | `sk_live_...` |
| Generic API Key | secret | `api_key=...` |
| Private Key | secret | `-----BEGIN RSA PRIVATE KEY-----` |
| JWT | credential | `eyJ...` |
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
  # Disable patterns you don't need
  disabled:
    - phone-us
    - ipv4-address

  # Add your own patterns
  custom:
    - name: internal-token
      category: secret
      regex: "INTERNAL_[A-Z0-9]{32}"
      description: "Internal API tokens"

# Values that should never be masked
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

Audit logs track detection counts only (never PII values):
```json
{
  "session_id": "20260308-143022",
  "command": "proxy",
  "detections": {"email": 3, "aws-access-key": 1},
  "total_sanitized": 4
}
```

## Limitations

- **Response rehydration** — Claude API uses SSE streaming for responses. Rehydrating placeholders in streaming responses is not yet supported. You'll see `[EMAIL_1]` in AI responses instead of the original value.
- **macOS/Linux only** — Windows is not supported yet (PTY and Keychain differences).

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
