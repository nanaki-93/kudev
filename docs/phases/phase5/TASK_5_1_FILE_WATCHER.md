# Task 5.1: Implement File Watcher

## Overview

This task implements **file system watching** using the fsnotify library to detect source code changes.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: None  
**Files to Create**:
- `pkg/watch/watcher.go` — File watcher implementation
- `pkg/watch/watcher_test.go` — Tests

---

## What You're Building

A file watcher that:
1. **Monitors** source directory recursively
2. **Filters** out excluded paths (.git, node_modules)
3. **Reports** file changes via channel
4. **Handles** context cancellation
5. **Recovers** from watcher errors

---

## Complete Implementation

```go
// pkg/watch/watcher.go

package watch

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "time"
    
    "github.com/fsnotify/fsnotify"
    
    "github.com/your-org/kudev/pkg/logging"
)

// FileChangeEvent represents a file system change.
type FileChangeEvent struct {
    // Path is the relative path of the changed file.
    Path string
    
    // Op is the operation type (write, create, delete, rename).
    Op string
    
    // Timestamp is when the event occurred.
    Timestamp time.Time
}

// Watcher monitors a directory for file changes.
type Watcher interface {
    // Watch starts watching for file changes.
    // Returns a channel that receives change events.
    // Closes the channel when context is cancelled.
    Watch(ctx context.Context, sourceDir string) (<-chan FileChangeEvent, error)
    
    // Close stops the watcher and releases resources.
    Close() error
}

// FSWatcher implements Watcher using fsnotify.
type FSWatcher struct {
    watcher    *fsnotify.Watcher
    exclusions []string
    logger     logging.Logger
}

// NewFSWatcher creates a new file system watcher.
func NewFSWatcher(exclusions []string, logger logging.Logger) (*FSWatcher, error) {
    w, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, fmt.Errorf("failed to create watcher: %w", err)
    }
    
    return &FSWatcher{
        watcher:    w,
        exclusions: append(defaultExclusions, exclusions...),
        logger:     logger,
    }, nil
}

// defaultExclusions are always ignored.
var defaultExclusions = []string{
    ".git",
    ".gitignore",
    ".kudev.yaml",
    ".kudev",
    "node_modules",
    "vendor",
    "__pycache__",
    ".pytest_cache",
    ".DS_Store",
    "Thumbs.db",
    ".idea",
    ".vscode",
    "*.swp",
    "*.swo",
    "*.log",
    "*.tmp",
}

// Watch starts watching the source directory.
func (w *FSWatcher) Watch(ctx context.Context, sourceDir string) (<-chan FileChangeEvent, error) {
    // Add directories recursively
    if err := w.addDirectoriesRecursively(sourceDir); err != nil {
        return nil, fmt.Errorf("failed to add directories: %w", err)
    }
    
    events := make(chan FileChangeEvent)
    
    go w.processEvents(ctx, sourceDir, events)
    
    w.logger.Info("watching for changes",
        "directory", sourceDir,
    )
    
    return events, nil
}

// addDirectoriesRecursively adds all non-excluded directories to the watcher.
func (w *FSWatcher) addDirectoriesRecursively(root string) error {
    return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        
        // Only watch directories
        if !info.IsDir() {
            return nil
        }
        
        // Get relative path
        relPath, err := filepath.Rel(root, path)
        if err != nil {
            return err
        }
        
        // Check exclusions
        if w.shouldExclude(relPath) {
            return filepath.SkipDir
        }
        
        // Add to watcher
        if err := w.watcher.Add(path); err != nil {
            return fmt.Errorf("failed to watch %s: %w", path, err)
        }
        
        w.logger.Debug("watching directory", "path", relPath)
        
        return nil
    })
}

// processEvents reads from fsnotify and sends to output channel.
func (w *FSWatcher) processEvents(ctx context.Context, sourceDir string, out chan<- FileChangeEvent) {
    defer close(out)
    
    for {
        select {
        case <-ctx.Done():
            return
            
        case event, ok := <-w.watcher.Events:
            if !ok {
                return
            }
            
            // Get relative path
            relPath, err := filepath.Rel(sourceDir, event.Name)
            if err != nil {
                continue
            }
            
            // Check exclusions
            if w.shouldExclude(relPath) {
                continue
            }
            
            // Convert operation
            op := w.opToString(event.Op)
            if op == "" {
                continue
            }
            
            // Handle new directories
            if event.Op&fsnotify.Create != 0 {
                if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
                    w.watcher.Add(event.Name)
                    w.logger.Debug("watching new directory", "path", relPath)
                }
            }
            
            w.logger.Debug("file changed",
                "path", relPath,
                "op", op,
            )
            
            // Send event
            select {
            case out <- FileChangeEvent{
                Path:      relPath,
                Op:        op,
                Timestamp: time.Now(),
            }:
            case <-ctx.Done():
                return
            }
            
        case err, ok := <-w.watcher.Errors:
            if !ok {
                return
            }
            w.logger.Error("watcher error", "error", err)
        }
    }
}

// shouldExclude checks if a path should be ignored.
func (w *FSWatcher) shouldExclude(relPath string) bool {
    // Normalize path
    relPath = filepath.ToSlash(relPath)
    
    // Skip current directory
    if relPath == "." {
        return false
    }
    
    // Get path components
    parts := strings.Split(relPath, "/")
    
    for _, exclusion := range w.exclusions {
        // Check each path component
        for _, part := range parts {
            if part == exclusion {
                return true
            }
            
            // Check glob patterns
            if matched, _ := filepath.Match(exclusion, part); matched {
                return true
            }
        }
    }
    
    return false
}

// opToString converts fsnotify operation to string.
func (w *FSWatcher) opToString(op fsnotify.Op) string {
    switch {
    case op&fsnotify.Write != 0:
        return "write"
    case op&fsnotify.Create != 0:
        return "create"
    case op&fsnotify.Remove != 0:
        return "delete"
    case op&fsnotify.Rename != 0:
        return "rename"
    default:
        return ""
    }
}

// Close stops the watcher.
func (w *FSWatcher) Close() error {
    return w.watcher.Close()
}

// Ensure FSWatcher implements Watcher
var _ Watcher = (*FSWatcher)(nil)
```

---

## Key Implementation Details

### 1. Directory-Level Watching

Watch directories, not individual files:
```go
filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
    if info.IsDir() {
        watcher.Add(path)
    }
    return nil
})
```

**Why?**
- Fewer file descriptors
- Automatically catches new files
- Avoids "too many open files" error

### 2. Skip Excluded Directories Early

```go
if w.shouldExclude(relPath) {
    return filepath.SkipDir  // Don't descend into .git, node_modules
}
```

### 3. Handle New Directories

Watch newly created directories:
```go
if event.Op&fsnotify.Create != 0 {
    if info, _ := os.Stat(event.Name); info.IsDir() {
        watcher.Add(event.Name)
    }
}
```

### 4. Non-Blocking Event Send

```go
select {
case out <- event:
case <-ctx.Done():
    return
}
```

---

## Testing

```go
// pkg/watch/watcher_test.go

package watch

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
)

type mockLogger struct{}

func (m *mockLogger) Info(msg string, kv ...interface{})  {}
func (m *mockLogger) Debug(msg string, kv ...interface{}) {}
func (m *mockLogger) Error(msg string, kv ...interface{}) {}

func TestFSWatcher_DetectsFileChange(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create initial file
    testFile := filepath.Join(tmpDir, "test.go")
    os.WriteFile(testFile, []byte("package main"), 0644)
    
    // Create watcher
    watcher, err := NewFSWatcher(nil, &mockLogger{})
    if err != nil {
        t.Fatalf("NewFSWatcher failed: %v", err)
    }
    defer watcher.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    events, err := watcher.Watch(ctx, tmpDir)
    if err != nil {
        t.Fatalf("Watch failed: %v", err)
    }
    
    // Modify file
    time.Sleep(100 * time.Millisecond) // Let watcher start
    os.WriteFile(testFile, []byte("package main\n// modified"), 0644)
    
    // Wait for event
    select {
    case event := <-events:
        if event.Path != "test.go" {
            t.Errorf("wrong path: %s", event.Path)
        }
        if event.Op != "write" {
            t.Errorf("wrong op: %s", event.Op)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestFSWatcher_ExcludesGit(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create .git directory
    gitDir := filepath.Join(tmpDir, ".git")
    os.Mkdir(gitDir, 0755)
    
    watcher, _ := NewFSWatcher(nil, &mockLogger{})
    defer watcher.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    
    events, _ := watcher.Watch(ctx, tmpDir)
    
    // Modify file in .git
    time.Sleep(100 * time.Millisecond)
    os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)
    
    // Should NOT receive event
    select {
    case event := <-events:
        t.Errorf("should not receive event for .git: %+v", event)
    case <-time.After(500 * time.Millisecond):
        // Good - no event received
    }
}

func TestFSWatcher_DetectsNewFile(t *testing.T) {
    tmpDir := t.TempDir()
    
    watcher, _ := NewFSWatcher(nil, &mockLogger{})
    defer watcher.Close()
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    events, _ := watcher.Watch(ctx, tmpDir)
    
    // Create new file
    time.Sleep(100 * time.Millisecond)
    newFile := filepath.Join(tmpDir, "new.go")
    os.WriteFile(newFile, []byte("package main"), 0644)
    
    // Wait for event
    select {
    case event := <-events:
        if event.Op != "create" {
            t.Errorf("expected create, got %s", event.Op)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestShouldExclude(t *testing.T) {
    watcher := &FSWatcher{exclusions: defaultExclusions}
    
    tests := []struct {
        path     string
        excluded bool
    }{
        {".git", true},
        {".git/HEAD", true},
        {"src/.git", true},
        {"node_modules", true},
        {"node_modules/express/index.js", true},
        {"main.go", false},
        {"src/main.go", false},
        {"Dockerfile", false},
        {"test.log", true},
        {".DS_Store", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.path, func(t *testing.T) {
            result := watcher.shouldExclude(tt.path)
            if result != tt.excluded {
                t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, result, tt.excluded)
            }
        })
    }
}
```

---

## Dependencies

Add fsnotify to go.mod:
```bash
go get github.com/fsnotify/fsnotify
```

---

## Checklist for Task 5.1

- [ ] Create `pkg/watch/watcher.go`
- [ ] Define `FileChangeEvent` struct
- [ ] Define `Watcher` interface
- [ ] Implement `FSWatcher` struct
- [ ] Implement `NewFSWatcher()` constructor
- [ ] Implement `Watch()` method
- [ ] Implement `addDirectoriesRecursively()` helper
- [ ] Implement `processEvents()` goroutine
- [ ] Implement `shouldExclude()` helper
- [ ] Implement `Close()` method
- [ ] Handle new directory creation
- [ ] Create `pkg/watch/watcher_test.go`
- [ ] Test file change detection
- [ ] Test exclusion patterns
- [ ] Test new file detection
- [ ] Run `go test ./pkg/watch -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Watching individual files
```go
// Wrong - too many file descriptors
for _, file := range files {
    watcher.Add(file)
}

// Right - watch directories
watcher.Add(directory)
```

❌ **Mistake 2**: Blocking event channel
```go
// Wrong - blocks if receiver is slow
out <- event

// Right - non-blocking with context
select {
case out <- event:
case <-ctx.Done():
    return
}
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 5.2** → Implement Event Debouncing

---

## References

- [fsnotify Documentation](https://pkg.go.dev/github.com/fsnotify/fsnotify)
- [filepath.Walk](https://pkg.go.dev/path/filepath#Walk)

