# Task 6.6: Implement CI/CD Pipeline

## Overview

This task implements **GitHub Actions workflows** for automated testing and release.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: All previous tasks  
**Files to Create**:
- `.github/workflows/test.yml` — Test workflow
- `.github/workflows/release.yml` — Release workflow
- `.goreleaser.yml` — Release configuration
- Updated `Makefile` — Build/test commands

---

## Test Workflow

```yaml
# .github/workflows/test.yml

name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  unit:
    name: Unit Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run tests
        run: go test ./... -v -race -coverprofile=coverage.out
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: coverage.out

  integration:
    name: Integration Tests
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Build kudev
        run: go build -o kudev ./cmd/main.go
      
      - name: Create Kind cluster
        uses: helm/kind-action@v1
        with:
          cluster_name: kudev-test
      
      - name: Run integration tests
        run: go test ./test/integration/... -tags=integration -v

  build:
    name: Build
    runs-on: ${{ matrix.os }}
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Build
        run: go build -o kudev ./cmd/main.go
      
      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: kudev-${{ matrix.os }}
          path: kudev*
```

---

## Release Workflow

```yaml
# .github/workflows/release.yml

name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    name: Release
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## GoReleaser Configuration

```yaml
# .goreleaser.yml

version: 1

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - main: ./cmd/main.go
    binary: kudev
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/your-org/kudev/pkg/version.Version={{.Version}}
      - -X github.com/your-org/kudev/pkg/version.Commit={{.Commit}}
      - -X github.com/your-org/kudev/pkg/version.Date={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ .Tag }}-next"

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
```

---

## Makefile

```makefile
# Makefile

.PHONY: all build test lint clean install

# Variables
BINARY_NAME=kudev
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X github.com/your-org/kudev/pkg/version.Version=$(VERSION)"

all: lint test build

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/main.go

install: build
	mv $(BINARY_NAME) $(GOPATH)/bin/

test:
	go test ./... -v -race

test-coverage:
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-integration:
	go test ./test/integration/... -tags=integration -v

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html

# Development helpers
run:
	go run ./cmd/main.go

watch:
	go run ./cmd/main.go watch

# Docker helpers
docker-build:
	docker build -t kudev:dev .

# Release helpers
release-snapshot:
	goreleaser release --snapshot --clean

release-dry-run:
	goreleaser release --skip-publish --clean
```

---

## Version Package

```go
// pkg/version/version.go

package version

var (
    // Version is the semantic version (set by ldflags)
    Version = "dev"
    
    // Commit is the git commit SHA (set by ldflags)
    Commit = "unknown"
    
    // Date is the build date (set by ldflags)
    Date = "unknown"
)

// Info returns formatted version info
func Info() string {
    return Version
}

// Full returns full version info
func Full() string {
    return Version + " (" + Commit + ") built " + Date
}
```

---

## Version Command

```go
// cmd/commands/version.go

package commands

import (
    "fmt"
    
    "github.com/spf13/cobra"
    
    "github.com/your-org/kudev/pkg/version"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("kudev %s\n", version.Full())
    },
}

func init() {
    rootCmd.AddCommand(versionCmd)
}
```

---

## Checklist for Task 6.6

- [ ] Create `.github/workflows/test.yml`
- [ ] Add lint job
- [ ] Add unit test job with coverage
- [ ] Add integration test job with Kind
- [ ] Add multi-platform build job
- [ ] Create `.github/workflows/release.yml`
- [ ] Create `.goreleaser.yml`
- [ ] Configure builds for linux/darwin/windows
- [ ] Configure ldflags for version injection
- [ ] Update `Makefile`
- [ ] Add build, test, lint targets
- [ ] Add install target
- [ ] Update `pkg/version/version.go`
- [ ] Update version command
- [ ] Test workflows locally with `act`
- [ ] Test release with `goreleaser release --snapshot`

---

## Releasing

```bash
# Tag a new version
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions will:
# 1. Run tests
# 2. Build binaries for all platforms
# 3. Create GitHub release
# 4. Upload binaries and checksums
```

---

## Common Commands

```bash
# Run all checks locally
make all

# Run just tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Build binary
make build

# Install to GOPATH/bin
make install

# Test release
make release-dry-run
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Phase 6 is now complete! 🎉
3. **v1.0 Release Ready!**

---

## References

- [GitHub Actions](https://docs.github.com/en/actions)
- [GoReleaser](https://goreleaser.com/)
- [golangci-lint](https://golangci-lint.run/)

