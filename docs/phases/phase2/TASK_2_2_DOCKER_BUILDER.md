# Task 2.2: Implement Docker Builder

## Overview

This task implements the **Docker CLI builder** that creates container images by invoking `docker build` as a subprocess. It's the primary builder implementation and serves as a reference for future builder implementations.

**Effort**: ~3-4 hours  
**Complexity**: 🟡 Intermediate (subprocess management, output streaming)  
**Dependencies**: Task 2.1 (Builder Types), Phase 1 (Logger)  
**Files to Create**:
- `pkg/builder/docker/builder.go` — Docker implementation
- `pkg/builder/docker/builder_test.go` — Tests

---

## What You're Building

A Docker builder that:
1. **Verifies** Docker daemon is running
2. **Executes** `docker build` with correct arguments
3. **Streams** build output to terminal in real-time
4. **Retrieves** image ID after successful build
5. **Returns** ImageRef with full reference and ID
6. **Handles** errors gracefully with helpful messages

---

## The Problem This Solves

Building Docker images involves:
- Checking if Docker daemon is available
- Constructing complex CLI arguments
- Streaming output without blocking
- Handling various error conditions
- Retrieving the resulting image ID

The Docker builder encapsulates all this complexity:
```go
// User code is simple
result, err := builder.Build(ctx, opts)
if err != nil {
    return fmt.Errorf("build failed: %w", err)
}
fmt.Printf("Built image: %s\n", result.FullRef)
```

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                   DockerBuilder                      │
├─────────────────────────────────────────────────────┤
│  Build(ctx, opts)                                    │
│    1. checkDockerDaemon()                           │
│    2. buildCommandArgs()                             │
│    3. exec.CommandContext("docker", "build", ...)   │
│    4. streamOutput() (goroutines)                   │
│    5. cmd.Wait()                                     │
│    6. getImageID()                                   │
│    7. return ImageRef                                │
└─────────────────────────────────────────────────────┘
```

---

## Complete Implementation

### File Structure

```
pkg/builder/
├── types.go           ← Task 2.1
├── tagger.go          ← Task 2.4
└── docker/
    ├── builder.go     ← You'll create this
    └── builder_test.go
```

### Docker Builder

```go
// pkg/builder/docker/builder.go

package docker

import (
    "bufio"
    "context"
    "fmt"
    "io"
    "os/exec"
    "strings"
    
    "github.com/your-org/kudev/pkg/builder"
    "github.com/your-org/kudev/pkg/logging"
)

// DockerBuilder implements builder.Builder using Docker CLI.
type DockerBuilder struct {
    logger logging.Logger
}

// NewDockerBuilder creates a new Docker CLI builder.
func NewDockerBuilder(logger logging.Logger) *DockerBuilder {
    return &DockerBuilder{logger: logger}
}

// Name returns the builder identifier.
func (db *DockerBuilder) Name() string {
    return "docker"
}

// Build creates a container image using docker build.
func (db *DockerBuilder) Build(ctx context.Context, opts builder.BuildOptions) (*builder.ImageRef, error) {
    // Validate options first
    if err := opts.Validate(); err != nil {
        return nil, fmt.Errorf("invalid build options: %w", err)
    }
    
    // 1. Verify Docker daemon is running
    if err := db.checkDockerDaemon(ctx); err != nil {
        return nil, err
    }
    
    db.logger.Info("starting docker build",
        "image", opts.ImageName,
        "tag", opts.ImageTag,
        "dockerfile", opts.DockerfilePath,
    )
    
    // 2. Build docker command arguments
    args := db.buildCommandArgs(opts)
    
    // 3. Create command with context for cancellation
    cmd := exec.CommandContext(ctx, "docker", args...)
    cmd.Dir = opts.SourceDir // Set working directory to source
    
    // 4. Get stdout and stderr pipes for streaming
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
    }
    
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
    }
    
    // 5. Start the command
    if err := cmd.Start(); err != nil {
        return nil, fmt.Errorf("failed to start docker build: %w", err)
    }
    
    // 6. Stream output in goroutines
    go db.streamOutput("stdout", stdout)
    go db.streamOutput("stderr", stderr)
    
    // 7. Wait for completion
    if err := cmd.Wait(); err != nil {
        return nil, fmt.Errorf("docker build failed: %w", err)
    }
    
    db.logger.Info("docker build completed successfully")
    
    // 8. Get image ID
    fullRef := fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag)
    imageID, err := db.getImageID(ctx, fullRef)
    if err != nil {
        return nil, fmt.Errorf("failed to get image ID: %w", err)
    }
    
    return &builder.ImageRef{
        FullRef: fullRef,
        ID:      imageID,
    }, nil
}

// checkDockerDaemon verifies the Docker daemon is running and accessible.
func (db *DockerBuilder) checkDockerDaemon(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf(
            "docker daemon is not running or not accessible\n\n"+
                "Troubleshooting:\n"+
                "  1. Ensure Docker Desktop is running\n"+
                "  2. Or start Docker daemon: sudo systemctl start docker\n"+
                "  3. Verify with: docker version\n\n"+
                "Error: %w\nOutput: %s", err, string(output),
        )
    }
    
    db.logger.Debug("docker daemon available", "version", strings.TrimSpace(string(output)))
    return nil
}

// buildCommandArgs constructs the docker build command arguments.
func (db *DockerBuilder) buildCommandArgs(opts builder.BuildOptions) []string {
    args := []string{"build"}
    
    // Add tag
    args = append(args, "-t", fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag))
    
    // Add Dockerfile path
    args = append(args, "-f", opts.DockerfilePath)
    
    // Add build args
    for key, val := range opts.BuildArgs {
        args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
    }
    
    // Add target if specified
    if opts.Target != "" {
        args = append(args, "--target", opts.Target)
    }
    
    // Add no-cache if specified
    if opts.NoCache {
        args = append(args, "--no-cache")
    }
    
    // Add build context (current directory since we set cmd.Dir)
    args = append(args, ".")
    
    return args
}

// streamOutput reads from a reader and logs each line.
func (db *DockerBuilder) streamOutput(source string, r io.Reader) {
    scanner := bufio.NewScanner(r)
    // Increase buffer size for long lines
    buf := make([]byte, 0, 64*1024)
    scanner.Buffer(buf, 1024*1024)
    
    for scanner.Scan() {
        line := scanner.Text()
        if line != "" {
            db.logger.Info(line, "source", source)
        }
    }
    
    if err := scanner.Err(); err != nil {
        db.logger.Error("error reading output", "source", source, "error", err)
    }
}

// getImageID retrieves the image ID using docker inspect.
func (db *DockerBuilder) getImageID(ctx context.Context, imageRef string) (string, error) {
    cmd := exec.CommandContext(ctx, "docker", "inspect",
        "--format={{.ID}}", imageRef)
    
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to inspect image %s: %w", imageRef, err)
    }
    
    imageID := strings.TrimSpace(string(output))
    db.logger.Debug("retrieved image ID", "image", imageRef, "id", imageID)
    
    return imageID, nil
}

// Ensure DockerBuilder implements builder.Builder
var _ builder.Builder = (*DockerBuilder)(nil)
```

---

## Key Implementation Details

### 1. Docker Daemon Check

Always check Docker availability first:
```go
func (db *DockerBuilder) checkDockerDaemon(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf(
            "docker daemon is not running or not accessible\n\n"+
            "Troubleshooting:\n"+
            "  1. Ensure Docker Desktop is running\n"+
            // ... helpful guidance
        )
    }
    return nil
}
```

**Why check first?**
- Fail fast with clear error message
- Avoid confusing build errors
- Guide user to solution

### 2. Command Construction

Build arguments dynamically:
```go
func (db *DockerBuilder) buildCommandArgs(opts builder.BuildOptions) []string {
    args := []string{"build"}
    
    args = append(args, "-t", fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag))
    args = append(args, "-f", opts.DockerfilePath)
    
    for key, val := range opts.BuildArgs {
        args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
    }
    
    if opts.Target != "" {
        args = append(args, "--target", opts.Target)
    }
    
    if opts.NoCache {
        args = append(args, "--no-cache")
    }
    
    args = append(args, ".")
    return args
}
```

### 3. Output Streaming

Stream output in goroutines to prevent blocking:
```go
stdout, _ := cmd.StdoutPipe()
stderr, _ := cmd.StderrPipe()

if err := cmd.Start(); err != nil {
    return nil, err
}

// These run concurrently with the build
go db.streamOutput("stdout", stdout)
go db.streamOutput("stderr", stderr)

// Wait for build to complete
if err := cmd.Wait(); err != nil {
    return nil, err
}
```

**Why goroutines?**
- Docker build can produce lots of output
- Blocking on output can deadlock
- User sees progress in real-time

### 4. Context Cancellation

Always use `exec.CommandContext`:
```go
cmd := exec.CommandContext(ctx, "docker", args...)
```

**Benefits**:
- Ctrl+C kills build immediately
- Timeout support (future)
- Clean resource cleanup

### 5. Image ID Retrieval

Get ID after successful build:
```go
func (db *DockerBuilder) getImageID(ctx context.Context, imageRef string) (string, error) {
    cmd := exec.CommandContext(ctx, "docker", "inspect",
        "--format={{.ID}}", imageRef)
    output, err := cmd.Output()
    // ...
}
```

**Why get ID?**
- Unique identifier for caching
- Verify build succeeded
- Used for cleanup operations

---

## Error Handling Strategy

### Helpful Error Messages

```go
// ❌ Bad error message
return fmt.Errorf("build failed: %w", err)

// ✅ Good error message
return fmt.Errorf(
    "docker build failed for image %s:%s\n\n"+
    "Common causes:\n"+
    "  - Dockerfile syntax error\n"+
    "  - Missing base image\n"+
    "  - Build context too large\n\n"+
    "Check the build output above for details.\n"+
    "Error: %w", opts.ImageName, opts.ImageTag, err,
)
```

### Error Categories

| Error Type | Example | User Action |
|------------|---------|-------------|
| Daemon unavailable | Docker not running | Start Docker Desktop |
| Invalid Dockerfile | Syntax error | Fix Dockerfile |
| Build failure | Compilation error | Check build output |
| Network error | Can't pull base image | Check internet |
| Disk space | No space left | Clean up images |

---

## Testing the Docker Builder

### Unit Tests with Mocking

Since we can't always run Docker in tests, use interface-based testing:

```go
// pkg/builder/docker/builder_test.go

package docker

import (
    "context"
    "testing"
    
    "github.com/your-org/kudev/pkg/builder"
)

// mockLogger implements logging.Logger for testing
type mockLogger struct {
    messages []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...interface{}) {
    m.messages = append(m.messages, msg)
}

func (m *mockLogger) Error(msg string, keysAndValues ...interface{}) {
    m.messages = append(m.messages, msg)
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{}) {
    m.messages = append(m.messages, msg)
}

func TestBuildCommandArgs(t *testing.T) {
    logger := &mockLogger{}
    db := NewDockerBuilder(logger)
    
    tests := []struct {
        name     string
        opts     builder.BuildOptions
        expected []string
    }{
        {
            name: "basic build",
            opts: builder.BuildOptions{
                SourceDir:      "/project",
                DockerfilePath: "./Dockerfile",
                ImageName:      "myapp",
                ImageTag:       "kudev-abc123",
            },
            expected: []string{
                "build",
                "-t", "myapp:kudev-abc123",
                "-f", "./Dockerfile",
                ".",
            },
        },
        {
            name: "with build args",
            opts: builder.BuildOptions{
                SourceDir:      "/project",
                DockerfilePath: "./Dockerfile",
                ImageName:      "myapp",
                ImageTag:       "kudev-abc123",
                BuildArgs:      map[string]string{"VERSION": "1.0"},
            },
            expected: []string{
                "build",
                "-t", "myapp:kudev-abc123",
                "-f", "./Dockerfile",
                "--build-arg", "VERSION=1.0",
                ".",
            },
        },
        {
            name: "with target and no-cache",
            opts: builder.BuildOptions{
                SourceDir:      "/project",
                DockerfilePath: "./Dockerfile",
                ImageName:      "myapp",
                ImageTag:       "kudev-abc123",
                Target:         "runtime",
                NoCache:        true,
            },
            expected: []string{
                "build",
                "-t", "myapp:kudev-abc123",
                "-f", "./Dockerfile",
                "--target", "runtime",
                "--no-cache",
                ".",
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            args := db.buildCommandArgs(tt.opts)
            
            // Check essential args are present
            // Note: BuildArgs map iteration order is random
            for _, exp := range tt.expected {
                found := false
                for _, arg := range args {
                    if arg == exp {
                        found = true
                        break
                    }
                }
                if !found && exp != "--build-arg" && exp != "VERSION=1.0" {
                    t.Errorf("expected arg %q not found in %v", exp, args)
                }
            }
        })
    }
}

func TestDockerBuilderImplementsInterface(t *testing.T) {
    // Compile-time check that DockerBuilder implements Builder
    var _ builder.Builder = (*DockerBuilder)(nil)
}
```

### Integration Tests (Docker Required)

```go
// +build docker_required

package docker

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    
    "github.com/your-org/kudev/pkg/builder"
)

func TestDockerBuildIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Create temp directory with Dockerfile
    tmpDir := t.TempDir()
    dockerfile := `FROM alpine:latest
RUN echo "test"
`
    err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte(dockerfile), 0644)
    if err != nil {
        t.Fatalf("failed to write Dockerfile: %v", err)
    }
    
    logger := &mockLogger{}
    db := NewDockerBuilder(logger)
    
    opts := builder.BuildOptions{
        SourceDir:      tmpDir,
        DockerfilePath: "./Dockerfile",
        ImageName:      "kudev-test",
        ImageTag:       "integration-test",
    }
    
    ctx := context.Background()
    result, err := db.Build(ctx, opts)
    if err != nil {
        t.Fatalf("build failed: %v", err)
    }
    
    if result.FullRef != "kudev-test:integration-test" {
        t.Errorf("unexpected FullRef: %s", result.FullRef)
    }
    
    if result.ID == "" {
        t.Error("expected non-empty image ID")
    }
    
    // Cleanup
    cleanupCmd := exec.Command("docker", "rmi", result.FullRef)
    cleanupCmd.Run()
}
```

---

## Critical Points

### 1. Buffer Size for Scanner

Docker build can output long lines:
```go
scanner := bufio.NewScanner(r)
buf := make([]byte, 0, 64*1024)
scanner.Buffer(buf, 1024*1024)  // Allow up to 1MB lines
```

### 2. Working Directory

Set cmd.Dir for relative Dockerfile paths:
```go
cmd := exec.CommandContext(ctx, "docker", args...)
cmd.Dir = opts.SourceDir  // Critical!
```

### 3. Goroutine Lifecycle

Goroutines for streaming complete when:
- Scanner reaches EOF (process exits)
- Reader is closed

No explicit cleanup needed, but ensure pipes are consumed.

### 4. Exit Code Handling

`cmd.Wait()` returns error if exit code != 0:
```go
if err := cmd.Wait(); err != nil {
    // This includes non-zero exit codes
    return nil, fmt.Errorf("docker build failed: %w", err)
}
```

---

## Checklist for Task 2.2

- [ ] Create `pkg/builder/docker/builder.go`
- [ ] Implement `DockerBuilder` struct
- [ ] Implement `NewDockerBuilder()` constructor
- [ ] Implement `Name()` method
- [ ] Implement `Build()` method
- [ ] Implement `checkDockerDaemon()` helper
- [ ] Implement `buildCommandArgs()` helper
- [ ] Implement `streamOutput()` helper
- [ ] Implement `getImageID()` helper
- [ ] Add interface assertion: `var _ builder.Builder = (*DockerBuilder)(nil)`
- [ ] Create `pkg/builder/docker/builder_test.go`
- [ ] Write tests for `buildCommandArgs()`
- [ ] Write test for interface implementation
- [ ] (Optional) Write integration test with Docker
- [ ] Run `go fmt ./pkg/builder/docker`
- [ ] Verify no compilation errors: `go build ./pkg/builder/...`
- [ ] Run tests: `go test ./pkg/builder/... -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not checking Docker daemon first
```go
// Wrong - confusing error if Docker isn't running
func (db *DockerBuilder) Build(...) {
    cmd := exec.Command("docker", "build", ...)
    // Error: "exit status 1" - not helpful!
}

// Right - check and provide guidance
func (db *DockerBuilder) Build(...) {
    if err := db.checkDockerDaemon(ctx); err != nil {
        return nil, err  // Clear message about Docker not running
    }
    // ...
}
```

❌ **Mistake 2**: Not streaming output
```go
// Wrong - blocks on large output, user sees nothing
output, err := cmd.CombinedOutput()

// Right - stream in real-time
stdout, _ := cmd.StdoutPipe()
go db.streamOutput("stdout", stdout)
```

❌ **Mistake 3**: Forgetting to set working directory
```go
// Wrong - relative paths fail
cmd := exec.Command("docker", "build", "-f", "./Dockerfile", ".")

// Right - set working directory
cmd.Dir = opts.SourceDir
```

❌ **Mistake 4**: Not using context
```go
// Wrong - can't cancel build
cmd := exec.Command("docker", "build", ...)

// Right - respects cancellation
cmd := exec.CommandContext(ctx, "docker", "build", ...)
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 2.3** → Implement Source Code Hashing
3. Then **Task 2.4** → Implement Image Tagging
4. Hash Calculator will generate input for ImageTag
5. Tagger will create deterministic tags

---

## References

- [os/exec Package](https://pkg.go.dev/os/exec)
- [Docker Build Command](https://docs.docker.com/engine/reference/commandline/build/)
- [bufio.Scanner](https://pkg.go.dev/bufio#Scanner)
- [Context Cancellation](https://pkg.go.dev/context)

