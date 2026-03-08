VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY = bin/saola
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test clean install coverage release-dry-run

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/saola

test:
	go test -race -v ./...

clean:
	rm -rf bin/

install:
	go install $(LDFLAGS) ./cmd/saola

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

release-dry-run:
	goreleaser release --snapshot --clean
