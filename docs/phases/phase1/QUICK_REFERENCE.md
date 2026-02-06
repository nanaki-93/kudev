# Phase 1 Quick Reference Guide

## For Busy Developers

This is a **TL;DR** version of Phase 1. For full details, see individual task files.

---

## Task Sequence & Time Estimates

```
Task 1.1 (2h)  → Define types (.kudev.yaml structure)
Task 1.2 (3h)  → Validation (error checking)
Task 1.3 (3h)  → Config loader (discovery + loading)
Task 1.4 (2h)  → Context validator (K8s safety)
Task 1.5 (2h)  → CLI scaffolding (Cobra commands)
Task 1.6 (2h)  → Logging + tests (Klog + integration tests)
         ────────
Total: ~14-16 hours
```

---

## Core Concepts

### 1. Configuration System
- **Input**: `.kudev.yaml` file
- **Processing**: Load → Validate → Apply defaults
- **Output**: Go struct (`DeploymentConfig`)

### 2. Safety First
- **Whitelist K8s contexts** (prevent prod deploys)
- **Validate all config fields** (fail fast)
- **Clear error messages** (actionable)

### 3. CLI Structure
```
kudev [global-flags] <command> [args]
  --config <path>      Override config path
  --debug              Enable debug logging
  --force-context      Skip context validation

Commands:
  version              Show version
  init                 Create .kudev.yaml
  validate             Check configuration
  (up, down, logs... in later phases)
```

---

## File Map

| File | Purpose | Key Functions |
|------|---------|---|
| `pkg/config/types.go` | Types | `DeploymentConfig`, `DeploymentSpec`, `EnvVar` |
| `pkg/config/validation.go` | Validation | `Validate()`, `validateDNSName()` |
| `pkg/config/errors.go` | Errors | `ValidationError` |
| `pkg/config/loader.go` | Discovery | `LoadConfig()`, `FileConfigLoader` |
| `pkg/config/defaults.go` | Defaults | `ApplyDefaults()` |
| `pkg/kubeconfig/context.go` | K8s API | `LoadCurrentContext()` |
| `pkg/kubeconfig/validator.go` | Safety | `ContextValidator.Validate()` |
| `pkg/logging/logger.go` | Logging | `Init()`, `Get()`, `Info()`, `Error()` |
| `cmd/commands/root.go` | CLI | Root command + global setup |
| `cmd/commands/version.go` | CLI | `version` command |
| `cmd/commands/init.go` | CLI | `init` command |
| `cmd/commands/validate.go` | CLI | `validate` command |

---

## Key Decisions (Why?)

| Decision | Choice | Why |
|----------|--------|-----|
| Config format | YAML | K8s standard, human-readable |
| CLI framework | Cobra | K8s standard, rich features |
| Logging | Klog | K8s standard, structured |
| Context safety | Whitelist | Safe by default, prevent accidents |
| Validation | All errors at once | Better UX, fewer iterations |

---

## Pattern: Config Loading Flow

```
1. Discover
   - Check --config flag
   - Search CWD, parents, project root
   - Check ~/.kudev/config

2. Load
   - Read file
   - Parse YAML
   - Convert to Go struct

3. Validate
   - Check required fields
   - Validate formats (DNS names, ports)
   - Check file existence

4. Apply Defaults
   - namespace → "default"
   - replicas → 1
   - ports → 8080

5. Return
   - Full DeploymentConfig ready to use
```

---

## Pattern: CLI Command Structure

```go
var command = &cobra.Command{
    Use: "commandname",
    Short: "One-liner",
    Long: `Detailed description`,
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Get shared config (loaded in PersistentPreRun)
        cfg := GetLoadedConfig()
        
        // 2. Get validator
        validator := GetValidator()
        
        // 3. Do work
        
        // 4. Return error or nil
        return nil
    },
}
```

---

## Implementation Checklist

### Task 1.1: Types
```
[ ] types.go created
[ ] DeploymentConfig struct with tags
[ ] ConfigMetadata struct
[ ] DeploymentSpec struct (all fields)
[ ] EnvVar struct
[ ] NewDeploymentConfig() helper
[ ] Doc comments on all types/fields
[ ] Compiles: go build ./pkg/config
```

### Task 1.2: Validation
```
[ ] validation.go created
[ ] Validate() method
[ ] validateDNSName() function
[ ] validatePort() function
[ ] validateEnv() function
[ ] errors.go with ValidationError type
[ ] Test file with >80% coverage
[ ] Table-driven tests
```

### Task 1.3: Loader
```
[ ] loader.go with FileConfigLoader
[ ] discover() finds .kudev.yaml
[ ] LoadFromPath() loads from file
[ ] Save() writes to file
[ ] defaults.go with ApplyDefaults()
[ ] isProjectRoot() detects .git, go.mod, etc.
[ ] Test file with discovery tests
[ ] Handles relative/absolute paths
```

### Task 1.4: Context
```
[ ] context.go with LoadCurrentContext()
[ ] validator.go with ContextValidator
[ ] Whitelist default contexts
[ ] Pattern matching for kind-*, *-local*, etc
[ ] Error messages show available contexts
[ ] Test file with pattern matching tests
```

### Task 1.5: CLI
```
[ ] cmd/main.go entry point
[ ] cmd/commands/root.go with global setup
[ ] PersistentPreRun loads config
[ ] version command works
[ ] init command interactive
[ ] validate command works
[ ] Help text clear
[ ] Compiles: go build ./cmd
```

### Task 1.6: Logging & Tests
```
[ ] pkg/logging/logger.go
[ ] Init(debug bool) function
[ ] Get() returns logger
[ ] Info(), Error(), Debug(), Warn() helpers
[ ] integration/phase1_test.go
[ ] Full flow integration test
[ ] Discovery test
[ ] Validation error message test
[ ] Run: go test ./... -cover
```

---

## Common Commands

```bash
# Build
go build ./cmd

# Run
./kudev version
./kudev init
./kudev validate

# Test
go test ./... -v
go test ./... -coverprofile=coverage.out

# Coverage report
go tool cover -html=coverage.out

# Format
go fmt ./...

# Lint
golangci-lint run ./...
```

---

## Debugging Tips

### Config not found?
```
1. Check --config flag
2. Check current directory: ls -la .kudev.yaml
3. Check parent directories
4. Run with --debug to see search paths
```

### Validation fails?
```
1. Run: kudev validate --debug
2. Check error message - what field is invalid?
3. Run: go test ./pkg/config -v -run TestValidate
4. Look at validation_test.go for examples
```

### Context blocked?
```
1. Run: kubectl config current-context
2. Is it in the whitelist? (docker-desktop, minikube, kind-*, etc.)
3. Use: kudev --force-context <command>
4. Or: kubectl config use-context <safe-context>
```

### Tests failing?
```
1. Run with verbose: go test -v
2. Check error message
3. Look at test file for expected behavior
4. Use: go test -run TestNameHere -v
```

---

## What NOT to Do

❌ Don't use `int` for ports/replicas → use `int32` (K8s standard)  
❌ Don't forget YAML tags → `yaml:"fieldName"`  
❌ Don't validate one error at a time → collect and report all  
❌ Don't skip error context → use `fmt.Errorf("context: %w", err)`  
❌ Don't forget to apply defaults → do it before validation  
❌ Don't use fmt.Printf for logging → use logger.Info()  
❌ Don't allow production contexts by default → whitelist safety  

---

## Quick Examples

### Loading config
```go
cfg, err := config.LoadConfig(ctx, configPath)
if err != nil {
    log.Fatal(err)
}
```

### Validating config
```go
if err := cfg.Validate(ctx); err != nil {
    log.Fatal(err)
}
```

### Checking K8s context
```go
validator, _ := kubeconfig.NewContextValidator(forceFlag)
if err := validator.Validate(); err != nil {
    log.Fatal(err)
}
```

### Logging
```go
logger := logging.Get()
logger.Info("step complete", "name", value)
logger.Error(err, "failed", "context", info)
```

---

## Dependencies

```bash
# Add to go.mod
go get sigs.k8s.io/yaml
go get k8s.io/klog/v2
go get k8s.io/client-go/tools/clientcmd
go get github.com/spf13/cobra
```

---

## Success Criteria

- ✅ All 6 tasks implemented
- ✅ No errors: `go build ./...`
- ✅ All tests pass: `go test ./...`
- ✅ Coverage >80%
- ✅ Commands work: `kudev version`, `kudev init`, `kudev validate`
- ✅ Error messages are helpful
- ✅ Context validation works

---

## Help! I'm Stuck

1. **Read the full task document** (`TASK_1_X.md`)
2. **Look at the code examples** in the docs
3. **Check the test file** - tests show expected behavior
4. **Run the test** - `go test -v` shows failures
5. **Look at the checklist** - what step are you on?

Each task file has:
- Problem overview
- Complete code examples
- Testing strategies
- Common mistakes
- Step-by-step instructions

---

## Next Steps After Phase 1

Once complete, move to **Phase 2: Image Pipeline**

Phase 2 adds:
- Docker image building
- Image registry operations
- Build optimization
- Cache management

---

## Resources

| Topic | Link |
|-------|------|
| Cobra | https://cobra.dev/ |
| Client-go | https://github.com/kubernetes/client-go |
| Klog | https://pkg.go.dev/k8s.io/klog/v2 |
| Go Testing | https://golang.org/pkg/testing/ |
| YAML | https://pkg.go.dev/gopkg.in/yaml.v3 |

---

**Start with Task 1.1 in [INDEX.md](./INDEX.md)** 🚀


