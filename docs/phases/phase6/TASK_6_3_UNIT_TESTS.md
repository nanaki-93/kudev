# Task 6.3: Write Comprehensive Unit Tests

## Overview

This task focuses on **improving test coverage** across all packages using Go testing best practices.

**Effort**: ~4-6 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: All previous tasks  
**Files to Create/Update**:
- `pkg/*_test.go` files across all packages

---

## Testing Patterns

### 1. Table-Driven Tests

```go
func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        input   Config
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid config",
            input:   Config{Name: "myapp", Port: 8080},
            wantErr: false,
        },
        {
            name:    "missing name",
            input:   Config{Port: 8080},
            wantErr: true,
            errMsg:  "name is required",
        },
        {
            name:    "invalid port",
            input:   Config{Name: "myapp", Port: -1},
            wantErr: true,
            errMsg:  "port must be positive",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Validate()
            
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErr && tt.errMsg != "" {
                if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("error message = %q, want to contain %q", err.Error(), tt.errMsg)
                }
            }
        })
    }
}
```

### 2. Fake Client Pattern

```go
func TestDeployer_Upsert(t *testing.T) {
    // Create fake K8s client
    fakeClient := fake.NewSimpleClientset()
    
    renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
    deployer := NewKubernetesDeployer(fakeClient, renderer, &mockLogger{})
    
    opts := DeploymentOptions{
        Config:    testConfig(),
        ImageRef:  "myapp:v1",
        ImageHash: "abc123",
    }
    
    // Test upsert
    status, err := deployer.Upsert(context.Background(), opts)
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    
    // Verify deployment was created
    dep, err := fakeClient.AppsV1().Deployments("default").Get(
        context.Background(), "myapp", metav1.GetOptions{},
    )
    if err != nil {
        t.Fatalf("Deployment not created: %v", err)
    }
    
    if dep.Name != "myapp" {
        t.Errorf("name = %q, want %q", dep.Name, "myapp")
    }
}
```

### 3. Mock Interfaces

```go
// Mock logger
type mockLogger struct {
    messages []string
}

func (m *mockLogger) Info(msg string, kv ...interface{}) {
    m.messages = append(m.messages, msg)
}

func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Error(msg string, kv ...interface{}) {}

// Mock builder
type mockBuilder struct {
    buildCalled bool
    buildErr    error
    buildResult *builder.ImageRef
}

func (m *mockBuilder) Build(ctx context.Context, opts builder.BuildOptions) (*builder.ImageRef, error) {
    m.buildCalled = true
    if m.buildErr != nil {
        return nil, m.buildErr
    }
    return m.buildResult, nil
}

func (m *mockBuilder) Name() string { return "mock" }
```

### 4. Test Helpers

```go
// testdata/helpers.go

func testConfig() *config.DeploymentConfig {
    return &config.DeploymentConfig{
        Metadata: config.ConfigMetadata{Name: "test-app"},
        Spec: config.DeploymentSpec{
            Namespace:   "default",
            ServicePort: 8080,
            Replicas:    1,
        },
    }
}

func tempDirWithFile(t *testing.T, filename, content string) string {
    t.Helper()
    
    dir := t.TempDir()
    path := filepath.Join(dir, filename)
    if err := os.WriteFile(path, []byte(content), 0644); err != nil {
        t.Fatalf("failed to write file: %v", err)
    }
    return dir
}
```

---

## Coverage Targets by Package

### pkg/config (85%+)

```go
// pkg/config/types_test.go
func TestConfigMetadata_Validate(t *testing.T) { ... }
func TestDeploymentSpec_Validate(t *testing.T) { ... }
func TestDeploymentSpec_GetDefaults(t *testing.T) { ... }

// pkg/config/validation_test.go
func TestValidateConfig(t *testing.T) { ... }
func TestValidateAppName(t *testing.T) { ... }
func TestValidatePort(t *testing.T) { ... }

// pkg/config/loader_test.go
func TestFileConfigLoader_Load(t *testing.T) { ... }
func TestFileConfigLoader_FindConfigFile(t *testing.T) { ... }
```

### pkg/builder (75%+)

```go
// pkg/builder/types_test.go
func TestBuildOptions_Validate(t *testing.T) { ... }
func TestImageRef_String(t *testing.T) { ... }

// pkg/builder/tagger_test.go
func TestTagger_GenerateTag(t *testing.T) { ... }
func TestIsKudevTag(t *testing.T) { ... }
func TestParseTag(t *testing.T) { ... }

// pkg/builder/docker/builder_test.go
func TestBuildCommandArgs(t *testing.T) { ... }
```

### pkg/deployer (80%+)

```go
// pkg/deployer/types_test.go
func TestTemplateData_Validate(t *testing.T) { ... }
func TestNewTemplateData(t *testing.T) { ... }
func TestDeploymentStatus_IsReady(t *testing.T) { ... }

// pkg/deployer/renderer_test.go
func TestRenderer_RenderDeployment(t *testing.T) { ... }
func TestRenderer_RenderService(t *testing.T) { ... }

// pkg/deployer/deployer_test.go
func TestDeployer_Upsert_Create(t *testing.T) { ... }
func TestDeployer_Upsert_Update(t *testing.T) { ... }
func TestDeployer_Delete(t *testing.T) { ... }
func TestDeployer_Status(t *testing.T) { ... }
```

### pkg/hash (85%+)

```go
// pkg/hash/calculator_test.go
func TestCalculator_Deterministic(t *testing.T) { ... }
func TestCalculator_ChangesWithContent(t *testing.T) { ... }
func TestCalculator_ExcludesGit(t *testing.T) { ... }

// pkg/hash/exclusions_test.go
func TestShouldExclude(t *testing.T) { ... }
func TestLoadDockerignore(t *testing.T) { ... }
```

### pkg/watch (70%+)

```go
// pkg/watch/watcher_test.go
func TestFSWatcher_DetectsChange(t *testing.T) { ... }
func TestFSWatcher_Exclusions(t *testing.T) { ... }

// pkg/watch/debounce_test.go
func TestDebouncer_BatchesEvents(t *testing.T) { ... }
func TestDebouncer_ResetsTimer(t *testing.T) { ... }
```

---

## Running Tests

```bash
# Run all tests
go test ./... -v

# Run with race detection
go test ./... -race

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package
go test ./pkg/config/... -v

# Run specific test
go test ./pkg/config -run TestValidateConfig -v
```

---

## Checklist for Task 6.3

- [ ] Review existing tests in each package
- [ ] Add missing test cases
- [ ] Use table-driven tests
- [ ] Use fake clients for K8s tests
- [ ] Add test helpers
- [ ] Achieve coverage targets:
  - [ ] pkg/config: 85%+
  - [ ] pkg/deployer: 80%+
  - [ ] pkg/builder: 75%+
  - [ ] pkg/hash: 85%+
  - [ ] pkg/watch: 70%+
- [ ] Run `go test ./... -race`
- [ ] Generate coverage report
- [ ] Fix any failing tests

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 6.4** → Implement Integration Tests

---

## References

- [Go Testing](https://pkg.go.dev/testing)
- [Table-Driven Tests](https://go.dev/blog/subtests)
- [Fake Clientset](https://pkg.go.dev/k8s.io/client-go/kubernetes/fake)

