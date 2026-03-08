# Contributing to Saola Proxy

## Development Setup

```bash
git clone https://github.com/nguyennghia/saola-proxy.git
cd saola-proxy
go build ./cmd/saola        # build binary to ./saola
go test -race ./...         # run all tests
```

## Running Tests

```bash
make test       # race-detected test run
make coverage   # generate coverage report (coverage.out)
```

## Adding New Patterns

1. Open `internal/scanner/patterns-builtin.go`
2. Add a compiled `*regexp.Regexp` variable at the top of the file
3. Call `r.Register(Pattern{...})` inside `RegisterBuiltins`
4. Add a test case in `internal/scanner/scanner_test.go` covering at least one positive and one negative example
5. Document the new pattern in `internal/cli/init-cmd.go` (the `disabled` comment list) and `README.md`

## PR Process

1. Fork the repo and create a feature branch (`git checkout -b feat/my-pattern`)
2. Write tests first; keep coverage above existing baseline
3. Run `gofmt -w .` before committing
4. Open a PR against `main` with a clear description of what and why

## Code Style

- `gofmt` is mandatory — CI will fail without it
- Keep packages small and focused; avoid circular dependencies
- No PII/secrets in logs or audit files — use the `OnDetection` callback pattern
- Prefer table-driven tests
