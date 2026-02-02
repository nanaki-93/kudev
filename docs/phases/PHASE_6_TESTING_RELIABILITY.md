# Phase 6: Testing & Reliability

**Objective**: Ensure the CLI works correctly across different machines and K8s distributions. Implement comprehensive testing and robust error handling.

**Timeline**: 1-2 weeks  
**Difficulty**: 🟢 Beginner-Friendly (testing patterns, no new features)  
**Dependencies**: Phase 1-5 (all previous phases complete)

---

## 📋 Overview

This phase is about quality, not features:

1. **Error Handling** — Custom error types with user-friendly messages
2. **Unit Tests** — Comprehensive coverage with fake clients
3. **Integration Tests** — Real workflows with Kind cluster
4. **CI/CD Pipeline** — GitHub Actions for validation
5. **Debugging Helpers** — System info and troubleshooting

---

## 🎯 Error Handling Strategy

### Custom Error Types

Create domain-specific error types:

```go
// pkg/errors/errors.go

type KudevError interface {
    error
    ExitCode() int
    UserMessage() string
    SuggestedAction() string
}

type ConfigError struct {
    msg string
}

type KubeAuthError struct {
    msg string
}

type BuildError struct {
    msg string
}

type DeployError struct {
    msg string
}
```

### Error Mapping

Map K8s API errors to user-friendly messages:

| K8s Error | Root Cause | Suggestion |
|-----------|-----------|-----------|
| IsNotFound | Resource missing | "Run `kudev init` first" |
| IsUnauthorized | No cluster auth | "Check kubeconfig: `kubectl auth can-i create deployments`" |
| IsConflict | Name conflict | "Use different app name or namespace" |
| IsForbidden | Permission denied | "Verify RBAC permissions" |

### Root Command Error Handler

```go
// cmd/root.go

func execute() error {
    return rootCmd.Execute()
}

func main() {
    if err := execute(); err != nil {
        if kerr, ok := err.(errors.KudevError); ok {
            fmt.Fprintf(os.Stderr, "❌ Error: %s\n", kerr.UserMessage())
            fmt.Fprintf(os.Stderr, "💡 Try: %s\n", kerr.SuggestedAction())
            os.Exit(kerr.ExitCode())
        }
        
        fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
        os.Exit(1)
    }
}
```

---

## 📝 Core Tasks

### Task 6.1: Define Custom Error Types

**Files**:
- `pkg/errors/errors.go` — Error type definitions
- `pkg/errors/messages.go` — User messages and suggestions

**Error Categories**:
```go
// Config errors (exit code 2)
type ConfigError struct { ... }  // Config not found, invalid, missing fields

// Auth errors (exit code 3)
type KubeAuthError struct { ... }  // Kubeconfig not found, context not allowed

// Build errors (exit code 4)
type BuildError struct { ... }  // Docker daemon down, build failed

// Deploy errors (exit code 5)
type DeployError struct { ... }  // K8s API error, image not found

// Other errors (exit code 1)
type GeneralError struct { ... }
```

**Success Criteria**:
- ✅ All error paths use custom types
- ✅ Exit codes are consistent and meaningful
- ✅ Messages are specific and helpful
- ✅ Suggested actions are actionable

---

### Task 6.2: Implement Error Interception

**Files**:
- `cmd/errors.go` — Error formatting and handling

**Implementation**:
```go
// Wrap all errors from pkg/ with context
// Example:
if err != nil {
    return &errors.DeployError{
        msg: fmt.Sprintf("failed to deploy to namespace '%s': %v", ns, err),
        suggestion: "Verify namespace exists: `kubectl get namespaces`",
    }
}
```

**Success Criteria**:
- ✅ All errors caught and formatted consistently
- ✅ Stderr output includes suggestion
- ✅ Correct exit codes set

---

### Task 6.3: Write Comprehensive Unit Tests

**Files**:
- `pkg/*_test.go` files across all packages

**Test Patterns**:

```go
// Table-driven tests
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        wantErr bool
        errType string
    }{
        // test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test
        })
    }
}

// Fake client usage
func TestUpsert(t *testing.T) {
    fakeClientset := fake.NewSimpleClientset()
    deployer := NewKubernetesDeployer(fakeClientset, ...)
    
    // test upsert logic
}

// Mock interfaces
type MockBuilder struct {
    buildErr error
}

func (m *MockBuilder) Build(...) (*ImageRef, error) {
    return &ImageRef{...}, m.buildErr
}
```

**Coverage Targets**:
- `pkg/config`: 85%+
- `pkg/deployer`: 80%+
- `pkg/builder`: 75%+
- `pkg/logs`, `pkg/portfwd`, `pkg/watch`: 70%+
- **Overall**: 75%+

**Test Files to Create**:
- `pkg/config/*_test.go`
- `pkg/kubeconfig/*_test.go`
- `pkg/builder/*_test.go`
- `pkg/hash/*_test.go`
- `pkg/registry/*_test.go`
- `pkg/deployer/*_test.go`
- `pkg/logs/*_test.go`
- `pkg/portfwd/*_test.go`
- `pkg/watch/*_test.go`
- `cmd/*_test.go`

---

### Task 6.4: Implement Integration Tests

**Files**:
- `test/integration/setup_test.go` — Kind cluster setup
- `test/integration/workflow_test.go` — Full workflows
- `test/integration/builder_test.go` — Real builds

**Integration Test Pattern**:

```go
// +build integration

func TestUpDownWorkflow(t *testing.T) {
    // Setup Kind cluster
    cluster, err := setupKindCluster(t)
    if err != nil {
        t.Fatalf("failed to setup kind cluster: %v", err)
    }
    defer cluster.Cleanup()
    
    // Create test app
    testApp := createTestApp(t)
    defer testApp.Cleanup()
    
    // Test init
    err = runCommand("kudev", "init", "-n", cluster.Namespace)
    if err != nil {
        t.Fatalf("init failed: %v", err)
    }
    
    // Test up
    err = runCommand("kudev", "up")
    if err != nil {
        t.Fatalf("up failed: %v", err)
    }
    
    // Verify deployment
    // ...
    
    // Test down
    err = runCommand("kudev", "down")
    if err != nil {
        t.Fatalf("down failed: %v", err)
    }
}
```

**Success Criteria**:
- ✅ Tests run in CI with Kind cluster
- ✅ Full workflows tested (init → up → down)
- ✅ Tests cleanup on failure
- ✅ Tests work with different K8s versions

---

### Task 6.5: Create Debug Command

**Files**:
- `pkg/debug/debug.go` — Debug utilities
- `cmd/debug.go` — Debug command

**Debug Output**:
```bash
$ kudev debug
Kudev Debug Information
=======================
Version: v1.0.0
Go Version: go1.23.2
OS: linux
Architecture: amd64

Kubeconfig: ~/.kube/config
Current Context: docker-desktop
Namespace: default

Docker Version: Docker version 25.0.0
Docker Daemon: Running ✓

Kind Clusters: kind (v1.29.0)
Minikube: Not installed

Kudev Config: .kudev.yaml
Config Valid: ✓
```

**Success Criteria**:
- ✅ Shows system info
- ✅ Shows K8s context and version
- ✅ Shows Docker/Minikube/Kind availability
- ✅ Shows kudev config status

---

### Task 6.6: Implement CI/CD Pipeline

**Files**:
- `.github/workflows/test.yml` — Unit + integration tests
- `.github/workflows/release.yml` — Release automation
- `Makefile` — Local build/test commands

**Test Workflow**:
```yaml
name: Tests
on: [push, pull_request]

jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.23.2'
      - run: go test ./... -v -race -coverprofile=coverage.out
      - run: go tool cover -html=coverage.out

  integration:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - uses: helm/kind-action@v1.7.0  # Create Kind cluster
      - run: go test -v ./test/integration/... -tags=integration
```

**Release Workflow**:
```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - uses: goreleaser/goreleaser-action@v4
        with:
          args: release --clean
```

**Makefile**:
```makefile
.PHONY: test build clean

test:
	go test ./... -v -race -coverprofile=coverage.out
	go tool cover -html=coverage.out

build:
	go build -o kudev ./cmd/main.go

run:
	go run ./cmd/main.go

clean:
	rm -f kudev coverage.out

lint:
	golangci-lint run ./...

fmt:
	gofmt -w ./
```

---

## 🧪 Test Execution

**Unit Tests**:
```bash
make test
# or
go test ./... -v -race
```

**Integration Tests**:
```bash
kind create cluster --name kudev-test
go test -v ./test/integration/... -tags=integration
kind delete cluster --name kudev-test
```

**Coverage Report**:
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## ✅ Phase 6 Success Criteria

- ✅ Custom error types for all failure modes
- ✅ Root command handles errors uniformly
- ✅ Unit tests >80% coverage
- ✅ Integration tests verify workflows
- ✅ CI/CD pipeline working
- ✅ Releases automated
- ✅ Debug command provides diagnostics

---

## 🎓 Final Checklist

Before releasing v1.0:

- [ ] All phases implemented and tested
- [ ] Code coverage >75% overall
- [ ] All commands have help text
- [ ] README.md complete with examples
- [ ] CONTRIBUTING.md written
- [ ] LICENSE file added
- [ ] Releases built for Linux/macOS/Windows
- [ ] Tagged with semantic versioning (v1.0.0)

---

## 🎉 Success!

You now have a production-ready Kubernetes development CLI that is:
- ✅ **Clean**: Clear separation of concerns, interfaces throughout
- ✅ **Extensible**: Easy to add new builders, registries, templates
- ✅ **Well-Tested**: Unit tests with fakes, integration tests with real clusters
- ✅ **Following K8s Standards**: Uses Cobra, client-go, klog, fake clientsets
- ✅ **User-Friendly**: Clear errors, helpful suggestions, great DX

---

**Congratulations! You've completed the Kudev Roadmap! 🚀**
