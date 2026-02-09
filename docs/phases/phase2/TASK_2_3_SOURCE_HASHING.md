# Task 2.3: Implement Source Code Hashing

## Overview

This task implements **deterministic hash calculation** for source code. The hash is used to generate unique image tags that change only when source code changes, enabling efficient caching and ensuring Kubernetes pulls updated images.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly (file walking, hashing)  
**Dependencies**: None (pure Go, no external dependencies)  
**Files to Create**:
- `pkg/hash/calculator.go` — Hash calculation logic
- `pkg/hash/exclusions.go` — File exclusion patterns
- `pkg/hash/calculator_test.go` — Tests

---

## What You're Building

A hash calculator that:
1. **Walks** the source directory recursively
2. **Excludes** files matching patterns (.git, node_modules, etc.)
3. **Hashes** file paths and contents
4. **Returns** a deterministic 8-character hash
5. **Respects** .dockerignore patterns

---

## The Problem This Solves

### Why Hash-Based Tagging?

| Strategy | Problem |
|----------|---------|
| Always `:latest` | K8s won't pull if tag exists locally |
| Git commit SHA | Doesn't reflect uncommitted changes |
| Random/UUID | Rebuilds every time (slow) |
| Timestamp | Not deterministic, rebuilds on redeploy |

**Hash-based tagging solves all these**:
- ✅ Same source = same tag (deterministic)
- ✅ Changed source = new tag (forces pull)
- ✅ Uncommitted changes included
- ✅ Fast rebuilds when unchanged

### Example Flow

```
Source code unchanged:
  hash = a1b2c3d4
  tag = kudev-a1b2c3d4
  → Image exists, skip rebuild!

Source code changed:
  hash = e5f6g7h8 (different!)
  tag = kudev-e5f6g7h8
  → New tag, must rebuild
  → K8s sees new tag, pulls image
```

---

## Architecture

```
┌────────────────────────────────────────────────────┐
│                   Calculator                        │
├────────────────────────────────────────────────────┤
│  Calculate(ctx)                                     │
│    1. Walk source directory                         │
│    2. For each file:                                │
│       - Check if excluded                           │
│       - Hash: path + content                        │
│    3. Combine all hashes → SHA256                   │
│    4. Truncate to 8 chars                           │
│    5. Return hash string                            │
└────────────────────────────────────────────────────┘
```

---

## Complete Implementation

### File Structure

```
pkg/hash/
├── calculator.go       ← Main logic
├── exclusions.go       ← Exclusion patterns
└── calculator_test.go  ← Tests
```

### Calculator Implementation

```go
// pkg/hash/calculator.go

package hash

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
)

// Calculator computes deterministic hashes of source code.
type Calculator struct {
    sourceDir  string
    exclusions []string
}

// NewCalculator creates a new hash calculator.
// sourceDir is the root directory to hash.
// exclusions are additional patterns to skip (beyond defaults).
func NewCalculator(sourceDir string, exclusions []string) *Calculator {
    return &Calculator{
        sourceDir:  sourceDir,
        exclusions: exclusions,
    }
}

// Calculate computes the hash of all source files.
// Returns an 8-character hash string.
func (c *Calculator) Calculate(ctx context.Context) (string, error) {
    // Collect all file hashes
    var fileHashes []string
    
    // Walk the directory
    err := filepath.WalkDir(c.sourceDir, func(path string, d fs.DirEntry, err error) error {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        if err != nil {
            return err
        }
        
        // Get relative path for consistent hashing across machines
        relPath, err := filepath.Rel(c.sourceDir, path)
        if err != nil {
            return fmt.Errorf("failed to get relative path: %w", err)
        }
        
        // Skip directories but check if we should skip entire subtree
        if d.IsDir() {
            if c.shouldExclude(relPath) {
                return filepath.SkipDir
            }
            return nil
        }
        
        // Skip excluded files
        if c.shouldExclude(relPath) {
            return nil
        }
        
        // Hash the file
        hash, err := c.hashFile(path, relPath)
        if err != nil {
            return fmt.Errorf("failed to hash file %s: %w", relPath, err)
        }
        
        fileHashes = append(fileHashes, hash)
        return nil
    })
    
    if err != nil {
        return "", fmt.Errorf("failed to walk directory: %w", err)
    }
    
    if len(fileHashes) == 0 {
        return "", fmt.Errorf("no files found in %s (all excluded?)", c.sourceDir)
    }
    
    // Sort for determinism (filesystem order varies)
    sort.Strings(fileHashes)
    
    // Combine all file hashes into final hash
    finalHasher := sha256.New()
    for _, h := range fileHashes {
        io.WriteString(finalHasher, h)
    }
    
    fullHash := hex.EncodeToString(finalHasher.Sum(nil))
    
    // Return first 8 characters
    return fullHash[:8], nil
}

// hashFile computes the hash of a single file.
// Includes both path and content for complete uniqueness.
func (c *Calculator) hashFile(absPath, relPath string) (string, error) {
    hasher := sha256.New()
    
    // Include relative path in hash
    // This ensures renaming a file changes the hash
    io.WriteString(hasher, relPath)
    
    // Read and hash file content
    file, err := os.Open(absPath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    if _, err := io.Copy(hasher, file); err != nil {
        return "", err
    }
    
    return hex.EncodeToString(hasher.Sum(nil)), nil
}

// SourceDir returns the source directory being hashed.
func (c *Calculator) SourceDir() string {
    return c.sourceDir
}
```

### Exclusions Implementation

```go
// pkg/hash/exclusions.go

package hash

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
)

// defaultExclusions are patterns always excluded from hashing.
var defaultExclusions = []string{
    ".git",
    ".gitignore",
    ".kudev.yaml",
    ".kudev",
    "node_modules",
    "vendor",
    "__pycache__",
    ".pytest_cache",
    "*.log",
    "*.tmp",
    ".DS_Store",
    "Thumbs.db",
    ".idea",
    ".vscode",
    "*.swp",
    "*.swo",
    "coverage.out",
    "coverage.html",
}

// shouldExclude checks if a path should be excluded from hashing.
func (c *Calculator) shouldExclude(relPath string) bool {
    // Normalize path separators for cross-platform
    relPath = filepath.ToSlash(relPath)
    
    // Check against default exclusions
    for _, pattern := range defaultExclusions {
        if c.matchPattern(relPath, pattern) {
            return true
        }
    }
    
    // Check against custom exclusions
    for _, pattern := range c.exclusions {
        if c.matchPattern(relPath, pattern) {
            return true
        }
    }
    
    return false
}

// matchPattern checks if a path matches an exclusion pattern.
// Supports:
// - Exact directory names: ".git" matches ".git" and ".git/anything"
// - Glob patterns: "*.log" matches "debug.log"
// - Path patterns: "src/*.tmp" matches "src/file.tmp"
func (c *Calculator) matchPattern(relPath, pattern string) bool {
    // Normalize pattern
    pattern = filepath.ToSlash(pattern)
    
    // Get path components
    pathParts := strings.Split(relPath, "/")
    
    // Check if any path component matches exactly
    for _, part := range pathParts {
        if part == pattern {
            return true
        }
        
        // Check glob match on component
        if matched, _ := filepath.Match(pattern, part); matched {
            return true
        }
    }
    
    // Check full path glob match
    if matched, _ := filepath.Match(pattern, relPath); matched {
        return true
    }
    
    // Check if pattern matches start of path (for directories)
    if strings.HasPrefix(relPath, pattern+"/") {
        return true
    }
    
    return false
}

// LoadDockerignore reads exclusion patterns from .dockerignore file.
// Returns empty slice if file doesn't exist.
func LoadDockerignore(sourceDir string) ([]string, error) {
    dockerignorePath := filepath.Join(sourceDir, ".dockerignore")
    
    file, err := os.Open(dockerignorePath)
    if os.IsNotExist(err) {
        return nil, nil // No .dockerignore, not an error
    }
    if err != nil {
        return nil, err
    }
    defer file.Close()
    
    var patterns []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        
        // Skip empty lines and comments
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        
        patterns = append(patterns, line)
    }
    
    return patterns, scanner.Err()
}

// GetDefaultExclusions returns a copy of the default exclusion patterns.
func GetDefaultExclusions() []string {
    result := make([]string, len(defaultExclusions))
    copy(result, defaultExclusions)
    return result
}
```

---

## Key Implementation Details

### 1. Determinism Requirements

For the hash to be deterministic:

```go
// 1. Sort file hashes before combining
sort.Strings(fileHashes)

// 2. Use relative paths (not absolute)
relPath, _ := filepath.Rel(c.sourceDir, path)

// 3. Include path in file hash (renaming = new hash)
io.WriteString(hasher, relPath)
```

### 2. Directory Exclusion

Skip entire directories efficiently:

```go
if d.IsDir() {
    if c.shouldExclude(relPath) {
        return filepath.SkipDir  // Don't descend into .git, node_modules
    }
    return nil
}
```

### 3. Cross-Platform Paths

Normalize path separators:

```go
relPath = filepath.ToSlash(relPath)  // Always use forward slashes
```

### 4. Context Cancellation

Support interruption for large directories:

```go
select {
case <-ctx.Done():
    return ctx.Err()
default:
}
```

---

## Testing Strategy

### Unit Tests

```go
// pkg/hash/calculator_test.go

package hash

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestCalculate_Deterministic(t *testing.T) {
    // Create temp directory with files
    tmpDir := t.TempDir()
    
    // Create test files
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
    
    calc := NewCalculator(tmpDir, nil)
    ctx := context.Background()
    
    // Calculate hash twice
    hash1, err := calc.Calculate(ctx)
    if err != nil {
        t.Fatalf("first calculation failed: %v", err)
    }
    
    hash2, err := calc.Calculate(ctx)
    if err != nil {
        t.Fatalf("second calculation failed: %v", err)
    }
    
    // Should be identical
    if hash1 != hash2 {
        t.Errorf("hash not deterministic: %s != %s", hash1, hash2)
    }
    
    // Should be 8 characters
    if len(hash1) != 8 {
        t.Errorf("hash length = %d, want 8", len(hash1))
    }
}

func TestCalculate_ChangesWithContent(t *testing.T) {
    tmpDir := t.TempDir()
    mainFile := filepath.Join(tmpDir, "main.go")
    
    // Write initial content
    os.WriteFile(mainFile, []byte("package main"), 0644)
    
    calc := NewCalculator(tmpDir, nil)
    ctx := context.Background()
    
    hash1, _ := calc.Calculate(ctx)
    
    // Modify file
    os.WriteFile(mainFile, []byte("package main\n// modified"), 0644)
    
    hash2, _ := calc.Calculate(ctx)
    
    // Hash should change
    if hash1 == hash2 {
        t.Errorf("hash should change when content changes: %s == %s", hash1, hash2)
    }
}

func TestCalculate_ExcludesGit(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create test files
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    
    calc := NewCalculator(tmpDir, nil)
    ctx := context.Background()
    
    hash1, _ := calc.Calculate(ctx)
    
    // Add .git directory (should be excluded)
    gitDir := filepath.Join(tmpDir, ".git")
    os.Mkdir(gitDir, 0755)
    os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main"), 0644)
    
    hash2, _ := calc.Calculate(ctx)
    
    // Hash should NOT change (git is excluded)
    if hash1 != hash2 {
        t.Errorf("hash should not change for excluded files: %s != %s", hash1, hash2)
    }
}

func TestCalculate_IncludesPath(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create file with same content but different name
    os.WriteFile(filepath.Join(tmpDir, "file1.go"), []byte("content"), 0644)
    
    calc := NewCalculator(tmpDir, nil)
    ctx := context.Background()
    
    hash1, _ := calc.Calculate(ctx)
    
    // Rename file (same content, different path)
    os.Remove(filepath.Join(tmpDir, "file1.go"))
    os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("content"), 0644)
    
    hash2, _ := calc.Calculate(ctx)
    
    // Hash should change (path is different)
    if hash1 == hash2 {
        t.Errorf("hash should change when path changes: %s == %s", hash1, hash2)
    }
}

func TestCalculate_CustomExclusions(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create files
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test data"), 0644)
    
    ctx := context.Background()
    
    // Calculate without custom exclusions
    calc1 := NewCalculator(tmpDir, nil)
    hash1, _ := calc1.Calculate(ctx)
    
    // Calculate with custom exclusion for .txt files
    calc2 := NewCalculator(tmpDir, []string{"*.txt"})
    hash2, _ := calc2.Calculate(ctx)
    
    // Hashes should be different
    if hash1 == hash2 {
        t.Errorf("custom exclusion should affect hash")
    }
    
    // Now modify excluded file
    os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("modified"), 0644)
    hash3, _ := calc2.Calculate(ctx)
    
    // Hash should NOT change (file is excluded)
    if hash2 != hash3 {
        t.Errorf("excluded file change should not affect hash: %s != %s", hash2, hash3)
    }
}

func TestShouldExclude(t *testing.T) {
    calc := NewCalculator("/project", nil)
    
    tests := []struct {
        path     string
        expected bool
    }{
        {".git", true},
        {".git/HEAD", true},
        {"src/.git", true},
        {"node_modules", true},
        {"node_modules/express/index.js", true},
        {"main.go", false},
        {"src/main.go", false},
        {"debug.log", true},
        {"src/debug.log", true},
        {".DS_Store", true},
        {"src/.DS_Store", true},
        {"Dockerfile", false},
        {"README.md", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.path, func(t *testing.T) {
            result := calc.shouldExclude(tt.path)
            if result != tt.expected {
                t.Errorf("shouldExclude(%q) = %v, want %v", tt.path, result, tt.expected)
            }
        })
    }
}

func TestLoadDockerignore(t *testing.T) {
    tmpDir := t.TempDir()
    
    // Create .dockerignore
    dockerignore := `# Comment
.git
node_modules
*.log

# Build artifacts
dist/
`
    os.WriteFile(filepath.Join(tmpDir, ".dockerignore"), []byte(dockerignore), 0644)
    
    patterns, err := LoadDockerignore(tmpDir)
    if err != nil {
        t.Fatalf("LoadDockerignore failed: %v", err)
    }
    
    expected := []string{".git", "node_modules", "*.log", "dist/"}
    if len(patterns) != len(expected) {
        t.Errorf("got %d patterns, want %d", len(patterns), len(expected))
    }
    
    for i, p := range expected {
        if i >= len(patterns) || patterns[i] != p {
            t.Errorf("pattern[%d] = %q, want %q", i, patterns[i], p)
        }
    }
}

func TestLoadDockerignore_NotExists(t *testing.T) {
    tmpDir := t.TempDir()
    
    patterns, err := LoadDockerignore(tmpDir)
    if err != nil {
        t.Errorf("unexpected error: %v", err)
    }
    
    if patterns != nil {
        t.Errorf("expected nil patterns, got %v", patterns)
    }
}
```

---

## Performance Considerations

### Large Repositories

For repositories with many files:

```go
// 1. Skip directories early (don't descend into node_modules)
if d.IsDir() && c.shouldExclude(relPath) {
    return filepath.SkipDir
}

// 2. Use streaming hash (don't load entire file into memory)
if _, err := io.Copy(hasher, file); err != nil {
    return "", err
}

// 3. Support cancellation for long operations
select {
case <-ctx.Done():
    return ctx.Err()
default:
}
```

### Benchmarking

```go
func BenchmarkCalculate(b *testing.B) {
    // Create test directory with realistic content
    tmpDir := b.TempDir()
    for i := 0; i < 100; i++ {
        content := fmt.Sprintf("package p%d\n// content %d", i, i)
        filename := fmt.Sprintf("file%d.go", i)
        os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
    }
    
    calc := NewCalculator(tmpDir, nil)
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        calc.Calculate(ctx)
    }
}
```

---

## How Hash Connects to Other Tasks

```
Task 2.3 (Hash Calculator) ← You are here
    ↓
    │ Hash used by:
    │
    ├─► Task 2.4 (Tagger)
    │   - GenerateTag() uses hash to create "kudev-{hash}"
    │
    └─► Task 2.2 (Docker Builder)
        - Can skip build if hash unchanged
        - Uses hash for cache invalidation
```

---

## Checklist for Task 2.3

- [ ] Create `pkg/hash/calculator.go`
- [ ] Implement `Calculator` struct
- [ ] Implement `NewCalculator()` constructor
- [ ] Implement `Calculate()` method
- [ ] Implement `hashFile()` helper
- [ ] Implement `SourceDir()` getter
- [ ] Create `pkg/hash/exclusions.go`
- [ ] Define `defaultExclusions` list
- [ ] Implement `shouldExclude()` method
- [ ] Implement `matchPattern()` helper
- [ ] Implement `LoadDockerignore()` function
- [ ] Implement `GetDefaultExclusions()` function
- [ ] Create `pkg/hash/calculator_test.go`
- [ ] Write tests for determinism
- [ ] Write tests for content changes
- [ ] Write tests for exclusions
- [ ] Write tests for path inclusion
- [ ] Write tests for .dockerignore loading
- [ ] Run `go fmt ./pkg/hash`
- [ ] Verify compilation: `go build ./pkg/hash`
- [ ] Run tests: `go test ./pkg/hash -v`
- [ ] Check coverage: `go test ./pkg/hash -cover`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Not sorting file hashes
```go
// Wrong - filesystem order varies between runs/platforms
for _, h := range fileHashes {
    finalHasher.Write([]byte(h))
}

// Right - sort for determinism
sort.Strings(fileHashes)
for _, h := range fileHashes {
    io.WriteString(finalHasher, h)
}
```

❌ **Mistake 2**: Using absolute paths
```go
// Wrong - hash changes between machines
io.WriteString(hasher, "/Users/alice/project/main.go")

// Right - use relative paths
relPath, _ := filepath.Rel(c.sourceDir, path)
io.WriteString(hasher, relPath)
```

❌ **Mistake 3**: Not handling path separators
```go
// Wrong - different on Windows vs Unix
relPath := "src\main.go"  // Windows
relPath := "src/main.go"  // Unix

// Right - normalize
relPath = filepath.ToSlash(relPath)  // Always "src/main.go"
```

❌ **Mistake 4**: Loading entire file into memory
```go
// Wrong - crashes on large files
content, _ := os.ReadFile(path)
hasher.Write(content)

// Right - stream the file
file, _ := os.Open(path)
io.Copy(hasher, file)
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 2.4** → Implement Image Tagging
3. Tagger will use Calculator to generate tags
4. Then **Task 2.5** → Implement Registry Loading

---

## References

- [crypto/sha256 Package](https://pkg.go.dev/crypto/sha256)
- [filepath.WalkDir](https://pkg.go.dev/path/filepath#WalkDir)
- [.dockerignore Reference](https://docs.docker.com/engine/reference/builder/#dockerignore-file)
- [Go File I/O Best Practices](https://golang.org/doc/effective_go#allocation_make)

