# Phase 7: Build & Release

**Status**: `[ ]` Not Started
**Depends on**: All previous phases
**Estimated effort**: 0.5 day

## Context Links
- [Plan Overview](./plan.md)
- Previous: [Phase 6 - Audit Logging](./phase-06-audit-logging.md)

## Overview

Production build pipeline, cross-compilation, goreleaser config, comprehensive README, and GitHub release automation. Makes Saola installable via `go install`, Homebrew, or direct binary download.

## Key Insights

- GoReleaser handles cross-compilation, checksums, changelog, GitHub releases
- `go install github.com/user/saola-proxy/cmd/saola@latest` should work out of the box
- README is the primary marketing/docs surface for an open-source CLI tool
- CI: GitHub Actions for test + lint on PR, goreleaser on tag push

## Requirements

- [ ] GoReleaser config (`.goreleaser.yaml`)
- [ ] GitHub Actions CI workflow (test + lint)
- [ ] GitHub Actions release workflow (goreleaser on tag)
- [ ] Comprehensive README.md
- [ ] Contributing guidelines (`CONTRIBUTING.md`)
- [ ] Makefile enhancements (install, release-dry-run)
- [ ] Cross-compile targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64

## Architecture

```
.goreleaser.yaml       # GoReleaser config
.github/
  workflows/
    ci.yaml            # Test + lint on push/PR
    release.yaml       # GoReleaser on tag
Makefile               # Enhanced with install, release targets
README.md              # Comprehensive docs
CONTRIBUTING.md        # Contribution guide
```

## Related Code Files

| File | Purpose |
|------|---------|
| `.goreleaser.yaml` | Release automation |
| `.github/workflows/ci.yaml` | CI pipeline |
| `.github/workflows/release.yaml` | Release pipeline |
| `Makefile` | Build targets |
| `README.md` | Project docs |
| `CONTRIBUTING.md` | Contributor guide |

## Implementation Steps

1. Create `.goreleaser.yaml`:
   - Binary name: `saola`
   - Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
   - Ldflags: `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}}`
   - Archive format: tar.gz (linux), zip (darwin)
   - Checksum: sha256
   - Changelog: auto from conventional commits

2. Create `.github/workflows/ci.yaml`:
   - Trigger: push to main, pull requests
   - Matrix: Go 1.22.x
   - Steps: checkout, setup-go, cache, `make test`, `make lint`
   - Race detector enabled in tests

3. Create `.github/workflows/release.yaml`:
   - Trigger: push tag `v*`
   - Steps: checkout, setup-go, goreleaser release
   - Requires `GITHUB_TOKEN` (default)

4. Enhance `Makefile`:
   - `install`: `go install ./cmd/saola`
   - `release-dry-run`: `goreleaser release --snapshot --clean`
   - `coverage`: `go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out`

5. Write `README.md`:
   - Project description + motivation (privacy for AI coding assistants)
   - Quick start (install + `saola wrap claude`)
   - Features list
   - Configuration section with example YAML
   - Built-in patterns table
   - Architecture overview diagram
   - Contributing link
   - License

6. Write `CONTRIBUTING.md`:
   - Development setup
   - Running tests
   - PR process
   - Code style (gofmt, golangci-lint)
   - Adding new patterns

## Todo List

- [ ] `.goreleaser.yaml`
- [ ] `.github/workflows/ci.yaml`
- [ ] `.github/workflows/release.yaml`
- [ ] Enhance Makefile
- [ ] Write README.md
- [ ] Write CONTRIBUTING.md
- [ ] Test goreleaser dry run
- [ ] Verify CI passes
- [ ] Tag v0.1.0 and test release

## Success Criteria

1. `goreleaser release --snapshot --clean` produces binaries for all 4 targets
2. CI workflow passes on push to main
3. Tagged release creates GitHub release with binaries and checksums
4. `go install github.com/user/saola-proxy/cmd/saola@latest` works
5. README provides clear quick-start instructions
6. Binary size is reasonable (<15MB)

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| GoReleaser config issues | Low | Dry-run testing before real release |
| CI flaky tests | Medium | No external dependencies in tests, use test fixtures |
| Windows not supported (PTY) | Low | Document as Linux/macOS only for MVP, add Windows note to README |

## Security Considerations

- GitHub Actions uses minimal permissions (contents: write for releases)
- No secrets beyond default GITHUB_TOKEN
- Binary checksums published with releases for verification
- Dependencies audited via `go mod verify`

## Post-MVP Roadmap (Out of Scope)

These items are documented for future reference but explicitly NOT part of MVP:

1. **HTTPS Proxy Mode** - `elazarl/goproxy` based MITM proxy for HTTP_PROXY env var approach
2. **SSE-aware stream parsing** - Proper SSE boundary detection instead of line buffering
3. **Homebrew tap** - Formula for `brew install saola`
4. **Windows support** - ConPTY-based wrapper
5. **ML-based PII detection** - Integration with Presidio or custom models
6. **VS Code extension** - Auto-wrap AI assistant processes
7. **Pattern sharing** - Community pattern registry
