# Phase 1: Core Foundation - Complete Implementation Guide

## Welcome to Phase 1! 🚀

This folder contains **detailed implementation guides** for each task in Phase 1. Each file is a complete deep-dive with:
- Problem overview
- Architecture decisions
- Complete code implementations
- Testing strategies
- Critical points and common mistakes
- Checklist for completion

---

## Quick Navigation

### 📋 Tasks (in order)

1. **[TASK_1_1_CONFIG_TYPES.md](./TASK_1_1_CONFIG_TYPES.md)** — Define configuration types
   - Go structs matching YAML schema
   - K8s API conventions
   - Field documentation
   - ~2-3 hours effort

2. **[TASK_1_2_CONFIGURATION_VALIDATION.md](./TASK_1_2_CONFIGURATION_VALIDATION.md)** — Implement validation
   - Validation rules and constraints
   - Custom error types with examples
   - Table-driven tests
   - ~3-4 hours effort

3. **[TASK_1_3_CONFIGURATION_LOADER.md](./TASK_1_3_CONFIGURATION_LOADER.md)** — Config discovery & loading
   - File discovery algorithm
   - Project root detection
   - Default value application
   - ~3-4 hours effort

4. **[TASK_1_4_KUBECONFIG_VALIDATION.md](./TASK_1_4_KUBECONFIG_VALIDATION.md)** — K8s context safety
   - Kubeconfig reading
   - Context validation with whitelist
   - Pattern matching (wildcards)
   - ~2-3 hours effort

5. **[TASK_1_5_CLI_SCAFFOLDING.md](./TASK_1_5_CLI_SCAFFOLDING.md)** — Cobra CLI structure
   - Root command with global setup
   - Subcommands: version, init, validate
   - PersistentPreRun initialization pattern
   - ~2-3 hours effort

6. **[TASK_1_6_LOGGING_TESTING.md](./TASK_1_6_LOGGING_TESTING.md)** — Logging & integration tests
   - Klog setup and usage
   - Integration test patterns
   - Coverage reporting
   - ~2-3 hours effort

**Total Effort**: ~15-20 hours  
**Total Complexity**: 🟢 Beginner to 🟡 Intermediate

---

## Architecture Overview

```
┌─────────────────────────────────────────────────┐
│              CLI Layer (Cobra)                   │
│  (root command, flags, subcommands)              │
│                  Task 1.5                        │
└────────────────┬────────────────────────────────┘
                 │
    ┌────────────┼────────────┐
    │            │            │
    ▼            ▼            ▼
┌─────────┐ ┌──────────┐ ┌──────────┐
│ Config  │ │Context   │ │ Logging  │
│ System  │ │Validator │ │ (Klog)   │
│         │ │          │ │          │
│Task 1.1 │ │Task 1.4  │ │Task 1.6  │
│ 1.2     │ │          │ │          │
│ 1.3     │ └──────────┘ └──────────┘
└─────────┘
    ▲
    │
  Config Loading Flow:
  Discover → Load → Validate → Apply Defaults
```

### Component Interactions

```
User runs: kudev validate

1. CLI Layer (Task 1.5)
   cmd/root.go:PersistentPreRunE()
   
2. Config Loading (Task 1.3)
   config.LoadConfig() → FileConfigLoader.Load()
   
3. Config Validation (Task 1.2)
   cfg.Validate() → checks all fields
   
4. Context Validation (Task 1.4)
   ContextValidator.Validate() → checks K8s context
   
5. Logging (Task 1.6)
   logging.Get().Info() → logs each step

6. Back to CLI (Task 1.5)
   cmd/validate.go → displays result
```

---

## Key Patterns Used

### Pattern 1: File Discovery with Fallback

```
Check --config flag
    ↓ (if set)
Check current directory
    ↓ (if not found)
Walk up parent directories
    ↓ (if not found)
Check home directory
    ↓ (if not found)
Provide helpful error message
```

### Pattern 2: Validation Layers

```
Level 1: Type validation (Go compiler)
Level 2: Required fields (code check)
Level 3: Format validation (regex, ranges)
Level 4: Context validation (K8s API)
Level 5: File system validation (files exist)
```

### Pattern 3: Configuration Application

```
Load from file
    ↓
Parse YAML
    ↓
Apply defaults
    ↓
Validate structure
    ↓
Return to caller
```

### Pattern 4: CLI Command Hierarchy

```
PersistentPreRunE (global initialization)
    ↓
Individual command RunE
    ↓
Access shared state (config, validator)
```

---

## File Structure After Phase 1

```
kudev/
├── cmd/
│   ├── main.go                    ← Entry point
│   └── commands/
│       ├── root.go                ← Root command + global setup
│       ├── version.go             ← Version command
│       ├── init.go                ← Interactive init
│       └── validate.go            ← Validate command
├── pkg/
│   ├── config/                    ← Config management
│   │   ├── types.go               ← Task 1.1: Data types
│   │   ├── validation.go          ← Task 1.2: Validation rules
│   │   ├── errors.go              ← Task 1.2: Error types
│   │   ├── loader.go              ← Task 1.3: Config discovery
│   │   ├── defaults.go            ← Task 1.3: Default values
│   │   ├── types_test.go
│   │   ├── validation_test.go
│   │   └── loader_test.go
│   ├── kubeconfig/                ← K8s context management
│   │   ├── context.go             ← Task 1.4: Kubeconfig reading
│   │   ├── validator.go           ← Task 1.4: Context validation
│   │   └── validator_test.go
│   ├── logging/                   ← Logging
│   │   └── logger.go              ← Task 1.6: Klog setup
│   └── version/
│       └── version.go             ← Version info
├── test/
│   └── integration/
│       └── phase1_test.go         ← Task 1.6: Integration tests
├── docs/
│   └── phases/
│       └── phase1/
│           ├── TASK_1_1_CONFIG_TYPES.md
│           ├── TASK_1_2_CONFIGURATION_VALIDATION.md
│           ├── TASK_1_3_CONFIGURATION_LOADER.md
│           ├── TASK_1_4_KUBECONFIG_VALIDATION.md
│           ├── TASK_1_5_CLI_SCAFFOLDING.md
│           ├── TASK_1_6_LOGGING_TESTING.md
│           └── INDEX.md           ← You are here
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Dependencies & Imports

### Go Standard Library
- `context` — Context propagation
- `fmt` — Formatting
- `os` — File system operations
- `path/filepath` — Path utilities
- `regexp` — Regular expressions
- `strings` — String operations

### External Dependencies

```bash
# Add to go.mod

# Kubernetes client libraries (K8s standard)
go get sigs.k8s.io/yaml                    # YAML parsing
go get k8s.io/klog/v2                       # Logging
go get k8s.io/client-go/tools/clientcmd    # Kubeconfig reading

# CLI framework (K8s standard)
go get github.com/spf13/cobra               # CLI commands
```

### Import Statements

```go
// For config types and loading
import (
    "sigs.k8s.io/yaml"
    "github.com/yourusername/kudev/pkg/config"
)

// For kubeconfig reading
import (
    "k8s.io/client-go/tools/clientcmd"
    "github.com/yourusername/kudev/pkg/kubeconfig"
)

// For CLI
import (
    "github.com/spf13/cobra"
)

// For logging
import (
    "k8s.io/klog/v2"
)
```

---

## Testing Strategy

### Test Files

| File | Purpose | Coverage |
|------|---------|----------|
| `pkg/config/types_test.go` | Type marshaling | ~70% |
| `pkg/config/validation_test.go` | Validation rules | ~85%+ |
| `pkg/config/loader_test.go` | Config discovery | ~80%+ |
| `pkg/kubeconfig/validator_test.go` | Context validation | ~80%+ |
| `test/integration/phase1_test.go` | Full flow | ~70% |

### Test Coverage Targets

- **Config package**: >85%
- **Kubeconfig package**: >80%
- **Overall Phase 1**: >80%

### Running Tests

```bash
# All tests
go test ./... -v

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Specific package
go test ./pkg/config -v

# With output
go test ./... -v -run TestValidate
```

---

## Critical Decisions Explained

### 1. YAML for Configuration

**Why YAML not JSON/TOML?**
- Industry standard in K8s ecosystem
- Human-readable and editable
- Matches kubectl conventions
- Supports comments

### 2. Cobra for CLI

**Why Cobra not flag package?**
- Used by kubectl, Docker, Kubernetes ecosystem
- Rich subcommand support
- Standard plugin pattern
- Better help generation

### 3. Klog for Logging

**Why Klog not log package?**
- K8s standard (used by kubectl, API server)
- Structured logging
- Verbosity levels
- Compatible with K8s log aggregation

### 4. Whitelist for Context Safety

**Why whitelist not blacklist?**
- Safe by default
- Explicit opt-in with --force-context
- Prevents accidental production deployments
- Follows security best practices

### 5. Fail-on-all-errors for Validation

**Why report all errors at once?**
- Users fix multiple problems in one iteration
- Better user experience
- Reduces feedback loops
- Aligns with Go conventions

---

## Common Mistakes to Avoid

### Mistake 1: Wrong YAML Tag Format
```go
// ❌ Wrong
type Config struct {
    Name string  // Missing tags!
}

// ✅ Right
type Config struct {
    Name string `yaml:"name" json:"name"`
}
```

### Mistake 2: Using int instead of int32
```go
// ❌ Wrong - doesn't match K8s types
Replicas int

// ✅ Right
Replicas int32
```

### Mistake 3: Not Handling Errors Consistently
```go
// ❌ Wrong - loses error context
data, _ := os.ReadFile(path)

// ✅ Right - preserves error context
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("failed to read %s: %w", path, err)
}
```

### Mistake 4: Not Testing Error Cases
```go
// ❌ Wrong - only tests happy path
func TestLoad(t *testing.T) {
    cfg, _ := loader.Load()
    assert(cfg != nil)
}

// ✅ Right - tests errors too
func TestLoad_Success(t *testing.T) { ... }
func TestLoad_NotFound(t *testing.T) { ... }
func TestLoad_InvalidYAML(t *testing.T) { ... }
```

### Mistake 5: Validation Errors Without Examples
```go
// ❌ Wrong - not helpful
return fmt.Errorf("invalid port")

// ✅ Right - helpful with example
return fmt.Errorf(
    "invalid port %d (must be 1-65535)\n\n"+
    "Example:\n  localPort: 8080",
    port,
)
```

---

## Code Quality Standards

### Go Formatting
```bash
go fmt ./...       # Auto-format
goimports -w .     # Organize imports
```

### Linting
```bash
golangci-lint run ./...
```

### Tests
```bash
go test ./... -v -race -cover
```

### Coverage
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Target: >80% coverage on all packages

---

## Helpful Resources

### Kubernetes & Client-go
- [Client-go examples](https://github.com/kubernetes/client-go/tree/master/examples)
- [Kubeconfig handling](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
- [K8s API conventions](https://kubernetes.io/docs/reference/using-api/api-concepts/)

### Go Best Practices
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)

### CLI & Configuration
- [Cobra documentation](https://cobra.dev/)
- [Viper config management](https://github.com/spf13/viper)
- [YAML in Go](https://pkg.go.dev/gopkg.in/yaml.v3)

### Testing
- [Go testing best practices](https://golang.org/pkg/testing/)
- [Table-driven tests](https://golang.org/src/encoding/json/encoding_test.go)
- [Test coverage](https://golang.org/blog/cover)

---

## Implementation Timeline

**Recommended schedule** (if working part-time):

| Week | Tasks | Hours |
|------|-------|-------|
| 1 | Task 1.1 (types) + 1.2 (validation) | 5-7 |
| 2 | Task 1.3 (loader) + 1.4 (context) | 5-7 |
| 3 | Task 1.5 (CLI) + 1.6 (logging) | 4-5 |
| 4 | Integration, testing, refinement | 3-4 |

**Total**: ~15-20 hours over 3-4 weeks

---

## Phase 1 Completion Checklist

### Code Implementation
- [ ] All 6 tasks implemented
- [ ] No compiler errors: `go build ./...`
- [ ] All tests pass: `go test ./...`
- [ ] Coverage >80%: `go tool cover`
- [ ] Formatting correct: `go fmt ./...`
- [ ] No lint warnings: `golangci-lint run`

### Functionality
- [ ] Config types work (marshal/unmarshal)
- [ ] Validation catches all errors
- [ ] Config loader discovers files
- [ ] Context validator blocks unsafe contexts
- [ ] CLI commands all work
- [ ] Logging shows debug info with --debug

### Documentation
- [ ] All functions documented
- [ ] All types have doc comments
- [ ] Examples in comments
- [ ] README updated with Phase 1 features

### Testing
- [ ] Unit tests >80% coverage
- [ ] Integration tests pass
- [ ] Edge cases tested
- [ ] Error cases tested
- [ ] Manual testing works

---

## Next Phase

Once Phase 1 is complete, move to **Phase 2: Image Pipeline**

Phase 2 will add:
- Docker image building
- Image tagging and pushing
- Build context optimization
- Layer caching

See: `../PHASE_2_IMAGE_PIPELINE.md`

---

## Quick Links

- [Phase 1 Original](../PHASE_1_CORE_FOUNDATION.md) — Original detailed phase document
- [Implementation Guide](../IMPLEMENTATION_GUIDE.md) — Overall implementation strategy
- [Project README](../../README.md) — Project overview

---

## Getting Help

If you're stuck on a task:

1. **Re-read the task document** — Often contains the answer
2. **Look at the checklist** — What's missing?
3. **Check the examples** — Code snippets in the docs
4. **Review the testing section** — Tests show expected behavior
5. **Run the tests** — `go test -v` shows failures

Remember: These detailed docs are designed so you should be able to implement without external references. Everything you need is here!

---

**Ready to start? Begin with [TASK_1_1_CONFIG_TYPES.md](./TASK_1_1_CONFIG_TYPES.md)** 🚀


