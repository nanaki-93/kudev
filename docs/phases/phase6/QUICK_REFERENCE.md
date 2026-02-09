# Phase 6 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 6. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 6.1 (3h)  → Custom error types
Task 6.2 (2h)  → Error interception
Task 6.3 (6h)  → Unit tests
Task 6.4 (4h)  → Integration tests
Task 6.5 (2h)  → Debug command
Task 6.6 (3h)  → CI/CD pipeline
         ────────
Total: ~14-20 hours
```

---

## Core Concepts

### 1. Error Handling
- Custom error types with exit codes
- User-friendly messages
- Suggested actions

### 2. Testing
- Table-driven unit tests
- Fake K8s client
- Integration tests with Kind

### 3. CI/CD
- GitHub Actions
- GoReleaser for releases
- Multi-platform builds

---

## File Map

| File | Purpose |
|------|---------|
| `pkg/errors/errors.go` | Error type definitions |
| `pkg/errors/messages.go` | Error constructors |
| `pkg/debug/debug.go` | Debug info gathering |
| `cmd/commands/debug.go` | Debug command |
| `.github/workflows/test.yml` | Test workflow |
| `.github/workflows/release.yml` | Release workflow |
| `.goreleaser.yml` | Release config |
| `Makefile` | Build commands |

---

## Error Types

| Type | Exit Code | Use Case |
|------|-----------|----------|
| `ConfigError` | 2 | Config not found, invalid |
| `KubeAuthError` | 3 | Kubeconfig issues |
| `BuildError` | 4 | Docker/build failures |
| `DeployError` | 5 | K8s API errors |
| `WatchError` | 6 | File watcher issues |

---

## Key Patterns

### Custom Error

```go
type ConfigError struct {
    Message    string
    Suggestion string
    Cause      error
}

func (e *ConfigError) ExitCode() int { return 2 }
func (e *ConfigError) UserMessage() string { return e.Message }
func (e *ConfigError) SuggestedAction() string { return e.Suggestion }
```

### Error Handling

```go
func handleError(err error) int {
    if kerr, ok := err.(kudevErrors.KudevError); ok {
        fmt.Fprintf(os.Stderr, "❌ Error: %s\n", kerr.UserMessage())
        fmt.Fprintf(os.Stderr, "💡 Suggestion: %s\n", kerr.SuggestedAction())
        return kerr.ExitCode()
    }
    return 1
}
```

### Table-Driven Tests

```go
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   Config
        wantErr bool
    }{
        {"valid", Config{Name: "app"}, false},
        {"empty", Config{}, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
            }
        })
    }
}
```

---

## Implementation Checklist

### Task 6.1: Custom Errors
```
[ ] pkg/errors/errors.go
[ ] KudevError interface
[ ] ConfigError, BuildError, DeployError
[ ] Exit code constants
```

### Task 6.2: Error Handling
```
[ ] handleError() in root.go
[ ] SilenceUsage, SilenceErrors
[ ] Exit code propagation
```

### Task 6.3: Unit Tests
```
[ ] pkg/config/*_test.go (85%+)
[ ] pkg/deployer/*_test.go (80%+)
[ ] pkg/builder/*_test.go (75%+)
[ ] pkg/hash/*_test.go (85%+)
```

### Task 6.4: Integration Tests
```
[ ] test/integration/setup_test.go
[ ] test/integration/workflow_test.go
[ ] Build tags: //go:build integration
```

### Task 6.5: Debug Command
```
[ ] pkg/debug/debug.go
[ ] cmd/commands/debug.go
[ ] System info gathering
```

### Task 6.6: CI/CD
```
[ ] .github/workflows/test.yml
[ ] .github/workflows/release.yml
[ ] .goreleaser.yml
[ ] Makefile
```

---

## Common Commands

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run integration tests
make test-integration

# Lint
make lint

# Build
make build

# Test release
make release-dry-run
```

---

## Coverage Targets

| Package | Target |
|---------|--------|
| pkg/config | 85%+ |
| pkg/deployer | 80%+ |
| pkg/builder | 75%+ |
| pkg/hash | 85%+ |
| pkg/watch | 70%+ |
| **Overall** | **75%+** |

---

## Release Process

```bash
# Tag new version
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions automatically:
# - Runs tests
# - Builds binaries
# - Creates GitHub release
# - Uploads artifacts
```

---

## Final Checklist

Before v1.0 release:
- [ ] All tests pass
- [ ] Coverage >75%
- [ ] All commands have help
- [ ] README.md complete
- [ ] LICENSE file added
- [ ] CI/CD working
- [ ] Tag v1.0.0

---

## Congratulations! 🎉

You've completed all 6 phases of kudev development:

- ✅ **Phase 1**: Core Foundation (Config, CLI)
- ✅ **Phase 2**: Image Pipeline (Build, Hash, Load)
- ✅ **Phase 3**: Manifest Orchestration (Deploy)
- ✅ **Phase 4**: Developer Experience (Logs, Port Forward)
- ✅ **Phase 5**: Live Watcher (Hot Reload)
- ✅ **Phase 6**: Testing & Reliability (Quality)

**kudev is production-ready!** 🚀

