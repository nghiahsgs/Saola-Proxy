# Phase 1: Project Setup

**Status**: `[ ]` Not Started
**Depends on**: None
**Estimated effort**: 0.5 day

## Context Links
- [Plan Overview](./plan.md)
- Next: [Phase 2 - PII Scanner](./phase-02-pii-scanner-engine.md)

## Overview

Bootstrap Go module, establish project structure, create Makefile, wire up Cobra CLI skeleton with `wrap` and `version` commands. Sets foundation for all subsequent phases.

## Key Insights

- Cobra provides subcommand routing, flag parsing, help generation out of the box
- Keep `cmd/saola/main.go` thin - delegate to internal packages
- Makefile handles build, test, lint, clean targets
- Use Go 1.22+ for stdlib improvements (slog, etc.)

## Requirements

- [ ] Initialize Go module `github.com/user/saola-proxy`
- [ ] Create directory structure per project spec
- [ ] Cobra CLI with root command + `wrap` subcommand + `version` subcommand
- [ ] Makefile with `build`, `test`, `lint`, `clean` targets
- [ ] `.gitignore` for Go projects
- [ ] `LICENSE` file (MIT)
- [ ] Basic `README.md` with project description

## Architecture

```
cmd/saola/
  main.go              # Entry point, calls root command
internal/
  cli/
    root.go            # Root cobra command
    wrap.go            # "wrap" subcommand (stub)
    version.go         # "version" subcommand
  scanner/             # (empty, Phase 2)
  sanitizer/           # (empty, Phase 3)
  wrapper/             # (empty, Phase 4)
  config/              # (empty, Phase 5)
  audit/               # (empty, Phase 6)
```

## Related Code Files

| File | Purpose |
|------|---------|
| `cmd/saola/main.go` | Entry point |
| `internal/cli/root.go` | Root command setup |
| `internal/cli/wrap.go` | Wrap subcommand |
| `internal/cli/version.go` | Version output |
| `Makefile` | Build automation |

## Implementation Steps

1. `go mod init github.com/user/saola-proxy`
2. Create directory tree (`cmd/saola/`, `internal/cli/`, `internal/scanner/`, etc.)
3. Implement `cmd/saola/main.go` - calls `cli.Execute()`
4. Implement `internal/cli/root.go` - root cobra command with description
5. Implement `internal/cli/wrap.go` - `saola wrap <command>` stub that prints "not implemented"
6. Implement `internal/cli/version.go` - prints version injected via ldflags
7. Create `Makefile`:
   - `VERSION` var from git tag or "dev"
   - `build`: `go build -ldflags "-X main.version=$(VERSION)" -o bin/saola ./cmd/saola`
   - `test`: `go test ./...`
   - `lint`: `golangci-lint run`
   - `clean`: `rm -rf bin/`
8. Create `.gitignore` (bin/, *.exe, .DS_Store, .env)
9. Create MIT `LICENSE`
10. Create minimal `README.md`

## Todo List

- [ ] Go module init
- [ ] Directory structure
- [ ] main.go entry point
- [ ] Root cobra command
- [ ] Wrap subcommand stub
- [ ] Version subcommand
- [ ] Makefile
- [ ] .gitignore
- [ ] LICENSE
- [ ] README.md
- [ ] Verify `make build` produces binary
- [ ] Verify `./bin/saola --help` works
- [ ] Verify `./bin/saola version` works

## Success Criteria

1. `make build` produces `bin/saola` binary without errors
2. `./bin/saola --help` shows usage with `wrap` and `version` subcommands
3. `./bin/saola version` prints version string
4. `./bin/saola wrap echo hello` prints "not implemented" (placeholder)
5. `make test` passes (even if no tests yet)
6. All directories exist per project structure

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Go version mismatch | Low | Document minimum Go version in README |
| golangci-lint not installed | Low | Make lint target optional, document install |

## Security Considerations

- No secrets in repo (`.gitignore` covers `.env`)
- MIT license is permissive, appropriate for open-source privacy tool

## Next Steps

Proceed to [Phase 2 - PII Scanner Engine](./phase-02-pii-scanner-engine.md) to build the core detection logic.
