# Task 6.4: Implement Integration Tests

## Overview

This task implements **end-to-end integration tests** that test the full kudev workflow with a real Kubernetes cluster.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate  
**Dependencies**: All previous tasks  
**Files to Create**:
- `test/integration/setup_test.go` — Test setup and helpers
- `test/integration/workflow_test.go` — Full workflow tests
- `test/integration/builder_test.go` — Real build tests

---

## Test Setup

### Build Tags

Use build tags to separate integration tests:

```go
//go:build integration
// +build integration

package integration
```

Run with:
```bash
go test ./test/integration/... -tags=integration
```

### Test Helpers

```go
// test/integration/setup_test.go

//go:build integration
// +build integration

package integration

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"
)

// TestContext holds test state
type TestContext struct {
    T          *testing.T
    Namespace  string
    AppName    string
    ProjectDir string
    CleanupFns []func()
}

// NewTestContext creates a new test context
func NewTestContext(t *testing.T) *TestContext {
    return &TestContext{
        T:         t,
        Namespace: fmt.Sprintf("kudev-test-%d", time.Now().Unix()),
        AppName:   "test-app",
    }
}

// Cleanup runs all cleanup functions
func (tc *TestContext) Cleanup() {
    for i := len(tc.CleanupFns) - 1; i >= 0; i-- {
        tc.CleanupFns[i]()
    }
}

// AddCleanup registers a cleanup function
func (tc *TestContext) AddCleanup(fn func()) {
    tc.CleanupFns = append(tc.CleanupFns, fn)
}

// CreateTestProject creates a temporary project directory
func (tc *TestContext) CreateTestProject() string {
    dir := tc.T.TempDir()
    
    // Create main.go
    mainGo := `package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello from kudev test!")
    })
    fmt.Println("Starting server on :8080")
    http.ListenAndServe(":8080", nil)
}
`
    os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0644)
    
    // Create Dockerfile
    dockerfile := `FROM golang:1.23-alpine
WORKDIR /app
COPY . .
RUN go build -o server main.go
CMD ["./server"]
`
    os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dockerfile), 0644)
    
    // Create .kudev.yaml
    config := fmt.Sprintf(`apiVersion: kudev/v1
kind: Deployment
metadata:
  name: %s
spec:
  namespace: %s
  imageName: %s
  servicePort: 8080
  replicas: 1
`, tc.AppName, tc.Namespace, tc.AppName)
    os.WriteFile(filepath.Join(dir, ".kudev.yaml"), []byte(config), 0644)
    
    tc.ProjectDir = dir
    return dir
}

// RunKudev runs the kudev CLI with arguments
func (tc *TestContext) RunKudev(args ...string) error {
    cmd := exec.Command("kudev", args...)
    cmd.Dir = tc.ProjectDir
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

// KubectlGet runs kubectl get and returns output
func (tc *TestContext) KubectlGet(resource, name string) (string, error) {
    cmd := exec.Command("kubectl", "get", resource, name, 
        "-n", tc.Namespace, "-o", "jsonpath={.metadata.name}")
    out, err := cmd.Output()
    return string(out), err
}

// WaitForDeployment waits for deployment to be ready
func (tc *TestContext) WaitForDeployment(timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    
    for time.Now().Before(deadline) {
        cmd := exec.Command("kubectl", "get", "deployment", tc.AppName,
            "-n", tc.Namespace, "-o", "jsonpath={.status.readyReplicas}")
        out, err := cmd.Output()
        if err == nil && string(out) == "1" {
            return nil
        }
        time.Sleep(2 * time.Second)
    }
    
    return fmt.Errorf("timeout waiting for deployment")
}

// DeleteNamespace deletes the test namespace
func (tc *TestContext) DeleteNamespace() {
    exec.Command("kubectl", "delete", "namespace", tc.Namespace, "--ignore-not-found").Run()
}

// RequireKind ensures Kind cluster is available
func RequireKind(t *testing.T) {
    cmd := exec.Command("kind", "get", "clusters")
    if err := cmd.Run(); err != nil {
        t.Skip("Kind not available, skipping integration tests")
    }
}

// RequireDocker ensures Docker is running
func RequireDocker(t *testing.T) {
    cmd := exec.Command("docker", "info")
    if err := cmd.Run(); err != nil {
        t.Skip("Docker not running, skipping integration tests")
    }
}
```

---

## Workflow Tests

```go
// test/integration/workflow_test.go

//go:build integration
// +build integration

package integration

import (
    "testing"
    "time"
)

func TestFullWorkflow(t *testing.T) {
    RequireDocker(t)
    RequireKind(t)
    
    tc := NewTestContext(t)
    defer tc.Cleanup()
    
    // Create test project
    tc.CreateTestProject()
    
    // Cleanup namespace after test
    tc.AddCleanup(func() {
        tc.DeleteNamespace()
    })
    
    // Test: kudev up
    t.Run("kudev up", func(t *testing.T) {
        if err := tc.RunKudev("up", "--no-logs"); err != nil {
            t.Fatalf("kudev up failed: %v", err)
        }
        
        // Wait for deployment
        if err := tc.WaitForDeployment(2 * time.Minute); err != nil {
            t.Fatalf("deployment not ready: %v", err)
        }
        
        // Verify deployment exists
        name, err := tc.KubectlGet("deployment", tc.AppName)
        if err != nil {
            t.Fatalf("deployment not found: %v", err)
        }
        if name != tc.AppName {
            t.Errorf("deployment name = %q, want %q", name, tc.AppName)
        }
        
        // Verify service exists
        name, err = tc.KubectlGet("service", tc.AppName)
        if err != nil {
            t.Fatalf("service not found: %v", err)
        }
    })
    
    // Test: kudev status
    t.Run("kudev status", func(t *testing.T) {
        if err := tc.RunKudev("status"); err != nil {
            t.Fatalf("kudev status failed: %v", err)
        }
    })
    
    // Test: kudev down
    t.Run("kudev down", func(t *testing.T) {
        if err := tc.RunKudev("down", "--force"); err != nil {
            t.Fatalf("kudev down failed: %v", err)
        }
        
        // Verify deployment is gone
        _, err := tc.KubectlGet("deployment", tc.AppName)
        if err == nil {
            t.Error("deployment should be deleted")
        }
    })
}

func TestValidateCommand(t *testing.T) {
    tc := NewTestContext(t)
    defer tc.Cleanup()
    
    tc.CreateTestProject()
    
    if err := tc.RunKudev("validate"); err != nil {
        t.Errorf("kudev validate failed: %v", err)
    }
}

func TestInitCommand(t *testing.T) {
    tc := NewTestContext(t)
    defer tc.Cleanup()
    
    // Use empty directory
    tc.ProjectDir = t.TempDir()
    
    if err := tc.RunKudev("init", "-n", "test-app"); err != nil {
        t.Errorf("kudev init failed: %v", err)
    }
    
    // Verify config was created
    // ...
}
```

---

## Builder Tests

```go
// test/integration/builder_test.go

//go:build integration
// +build integration

package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/your-org/kudev/pkg/builder"
    "github.com/your-org/kudev/pkg/builder/docker"
    "github.com/your-org/kudev/pkg/hash"
)

func TestDockerBuild(t *testing.T) {
    RequireDocker(t)
    
    tc := NewTestContext(t)
    defer tc.Cleanup()
    
    tc.CreateTestProject()
    
    // Calculate hash
    calc := hash.NewCalculator(tc.ProjectDir, nil)
    tagger := builder.NewTagger(calc)
    tag, err := tagger.GenerateTag(context.Background(), false)
    if err != nil {
        t.Fatalf("failed to generate tag: %v", err)
    }
    
    // Build image
    db := docker.NewDockerBuilder(&testLogger{t})
    opts := builder.BuildOptions{
        SourceDir:      tc.ProjectDir,
        DockerfilePath: filepath.Join(tc.ProjectDir, "Dockerfile"),
        ImageName:      "kudev-test",
        ImageTag:       tag,
    }
    
    imageRef, err := db.Build(context.Background(), opts)
    if err != nil {
        t.Fatalf("build failed: %v", err)
    }
    
    if imageRef.FullRef == "" {
        t.Error("imageRef.FullRef is empty")
    }
    
    t.Logf("Built image: %s", imageRef.FullRef)
    
    // Cleanup: remove image
    tc.AddCleanup(func() {
        exec.Command("docker", "rmi", imageRef.FullRef).Run()
    })
}

func TestHashDeterminism(t *testing.T) {
    tc := NewTestContext(t)
    defer tc.Cleanup()
    
    tc.CreateTestProject()
    
    calc := hash.NewCalculator(tc.ProjectDir, nil)
    
    hash1, err := calc.Calculate(context.Background())
    if err != nil {
        t.Fatalf("first hash failed: %v", err)
    }
    
    hash2, err := calc.Calculate(context.Background())
    if err != nil {
        t.Fatalf("second hash failed: %v", err)
    }
    
    if hash1 != hash2 {
        t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
    }
}

type testLogger struct {
    t *testing.T
}

func (l *testLogger) Info(msg string, kv ...interface{})  { l.t.Log(msg) }
func (l *testLogger) Debug(msg string, kv ...interface{}) { l.t.Log(msg) }
func (l *testLogger) Error(msg string, kv ...interface{}) { l.t.Log(msg) }
```

---

## Running Integration Tests

```bash
# Create Kind cluster (if needed)
kind create cluster --name kudev-test

# Build kudev binary
go build -o kudev ./cmd/main.go

# Run integration tests
go test ./test/integration/... -tags=integration -v

# Cleanup
kind delete cluster --name kudev-test
```

---

## Checklist for Task 6.4

- [ ] Create `test/integration/setup_test.go`
- [ ] Implement `TestContext` helper
- [ ] Implement `CreateTestProject()` helper
- [ ] Implement `RunKudev()` helper
- [ ] Implement `WaitForDeployment()` helper
- [ ] Create `test/integration/workflow_test.go`
- [ ] Test full up/down workflow
- [ ] Test validate command
- [ ] Test init command
- [ ] Create `test/integration/builder_test.go`
- [ ] Test Docker build
- [ ] Test hash determinism
- [ ] Add build tags to all files
- [ ] Run tests with Kind cluster

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 6.5** → Create Debug Command

---

## References

- [Go Build Tags](https://pkg.go.dev/go/build#hdr-Build_Constraints)
- [Kind](https://kind.sigs.k8s.io/)

