# Phase 5: Config System

**Status**: `[ ]` Not Started
**Depends on**: Phase 1 (project structure), Phase 2 (scanner patterns)
**Estimated effort**: 0.5 day

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 4 - CLI Wrapper](./phase-04-cli-wrapper.md)
- Next: [Phase 6 - Audit Logging](./phase-06-audit-logging.md)

## Overview

YAML-based configuration system. Supports global config (`~/.saola/config.yaml`), project-level config (`.saola.yaml`), and CLI flag overrides. Manages pattern enable/disable, custom patterns, whitelist values, and general settings.

## Key Insights

- Config precedence: CLI flags > project `.saola.yaml` > global `~/.saola/config.yaml` > defaults
- XDG support: respect `$XDG_CONFIG_HOME` if set, else `~/.saola/`
- Keep config minimal for MVP - only settings that users actually need to tweak
- `saola init` command generates default config file with comments

## Requirements

- [ ] YAML config parsing with `gopkg.in/yaml.v3`
- [ ] Config struct with all settings
- [ ] Load order: defaults → global → project → CLI flags
- [ ] `saola init` command to generate default config
- [ ] Pattern enable/disable per config
- [ ] Custom pattern definitions in config
- [ ] Whitelist: values that should never be redacted
- [ ] XDG_CONFIG_HOME support

## Architecture

```
internal/config/
  config.go            # Config struct + Load() function
  defaults.go          # Default values
  config_test.go       # Tests
```

### Config File Format

```yaml
# ~/.saola/config.yaml
version: 1

# General settings
log_level: info        # debug, info, warn, error
audit_enabled: true

# Pattern configuration
patterns:
  # Disable specific built-in patterns
  disabled:
    - ip-address
    - phone-us

  # Custom patterns
  custom:
    - name: internal-api-key
      category: secret
      regex: "INTERNAL_[A-Z0-9]{32}"
      description: "Internal API keys"

# Values that should never be redacted
whitelist:
  - "127.0.0.1"
  - "localhost"
  - "0.0.0.0"
  - "example.com"
  - "test@example.com"
```

### Config Struct

```go
type Config struct {
    Version      int              `yaml:"version"`
    LogLevel     string           `yaml:"log_level"`
    AuditEnabled bool             `yaml:"audit_enabled"`
    Patterns     PatternConfig    `yaml:"patterns"`
    Whitelist    []string         `yaml:"whitelist"`
}

type PatternConfig struct {
    Disabled []string        `yaml:"disabled"`
    Custom   []CustomPattern `yaml:"custom"`
}

type CustomPattern struct {
    Name        string `yaml:"name"`
    Category    string `yaml:"category"`
    Regex       string `yaml:"regex"`
    Description string `yaml:"description"`
}
```

## Related Code Files

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Config loading + merging |
| `internal/config/defaults.go` | Default config values |
| `internal/config/config_test.go` | Tests |
| `internal/cli/init.go` | `saola init` command |
| `internal/scanner/registry.go` | Consumes config to enable/disable patterns |

## Implementation Steps

1. Create `defaults.go`:
   - `DefaultConfig() *Config` - returns config with all defaults
   - Default whitelist: `127.0.0.1`, `0.0.0.0`, `localhost`, `example.com`
2. Create `config.go`:
   - `Load() (*Config, error)`:
     a. Start with defaults
     b. Find global config: `$XDG_CONFIG_HOME/saola/config.yaml` or `~/.saola/config.yaml`
     c. Find project config: walk up from cwd looking for `.saola.yaml`
     d. Parse and merge each layer
     e. Return merged config
   - `configDir() string` - XDG-aware config directory
3. Create `internal/cli/init.go`:
   - `saola init` command
   - Writes default config to `~/.saola/config.yaml` with YAML comments
   - Prompts before overwriting existing config
4. Integrate config with scanner:
   - After loading config, disable patterns listed in `config.Patterns.Disabled`
   - Register custom patterns from `config.Patterns.Custom`
   - Apply whitelist: scanner skips matches whose value is in whitelist
5. Add `--config` flag to root command for explicit config path override
6. Create `config_test.go`:
   - Test default config values
   - Test YAML parsing
   - Test merge precedence (project overrides global)
   - Test XDG path resolution
   - Test invalid YAML handling
   - Test custom pattern validation (invalid regex)

## Todo List

- [ ] `defaults.go` - default config
- [ ] `config.go` - Load() with merge logic
- [ ] `config_test.go` - unit tests
- [ ] `init.go` - saola init command
- [ ] Integrate config with scanner registry
- [ ] Add whitelist filtering to scanner
- [ ] Add `--config` flag to root command
- [ ] Test with real config file

## Success Criteria

1. `saola init` creates `~/.saola/config.yaml` with sensible defaults and comments
2. Disabling a pattern in config prevents scanner from detecting it
3. Custom patterns in config are registered and functional
4. Whitelisted values pass through unsanitized
5. Project-level `.saola.yaml` overrides global config
6. Missing config file uses defaults without error
7. Invalid YAML produces clear error message

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Invalid custom regex crashes app | Medium | Validate regex at config load time, return error |
| Config file permissions too open | Low | Warn if config is world-readable (may contain custom patterns) |
| Breaking config format changes | Low | Version field in config for future migrations |

## Security Considerations

- Config file may contain custom regex patterns but should never contain PII
- Warn users not to put actual secrets in whitelist (only patterns like `example.com`)
- Config file permissions should be user-readable only (0600)

## Next Steps

Proceed to [Phase 6 - Audit Logging](./phase-06-audit-logging.md) for session statistics and logging.
