# Task 2.4: Implement Image Tagging

## Overview

This task implements **image tag generation** based on source code hash. The tagger creates deterministic tags that change only when source code changes, enabling efficient caching and forcing Kubernetes to pull updated images.

**Effort**: ~1-2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Task 2.3 (Hash Calculator)  
**Files to Create**:
- `pkg/builder/tagger.go` — Tag generation logic
- `pkg/builder/tagger_test.go` — Tests

---

## What You're Building

A tagger that:
1. **Uses** hash calculator to get source hash
2. **Generates** tag in format `kudev-{8-char-hash}`
3. **Optionally** adds timestamp for forced rebuilds
4. **Validates** tag format compliance

---

## The Problem This Solves

### Why Tagged Images?

Kubernetes image pulling behavior:

| Tag | imagePullPolicy: IfNotPresent | imagePullPolicy: Always |
|-----|------------------------------|------------------------|
| `myapp:latest` | Won't pull if exists | Pulls every time (slow) |
| `myapp:v1.0` | Won't pull if exists | Pulls every time |
| `myapp:kudev-a1b2c3d4` | Won't pull if exists | Pulls once, cached |

**Hash-based tags give us the best of both worlds**:
- New tag when code changes → K8s pulls new image
- Same tag when code unchanged → K8s uses cached image

### Tag Format

```
myapp:kudev-a1b2c3d4
      │     │
      │     └── 8-char source hash
      └──────── kudev prefix (identifies our images)
```

With timestamp (forced rebuild):
```
myapp:kudev-a1b2c3d4-20250209-143025
                     │
                     └── UTC timestamp (YYYYMMDD-HHMMSS)
```

---

## Architecture

```
┌────────────────────────────────────────────────────┐
│                     Tagger                          │
├────────────────────────────────────────────────────┤
│  GenerateTag(ctx, forceTimestamp)                  │
│    1. calculator.Calculate(ctx) → hash             │
│    2. format: "kudev-{hash}"                       │
│    3. if forceTimestamp: add "-{timestamp}"        │
│    4. return tag                                    │
└────────────────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────────────────┐
│                   Calculator                        │
│                   (Task 2.3)                        │
└────────────────────────────────────────────────────┘
```

---

## Complete Implementation

### File Structure

```
pkg/builder/
├── types.go           ← Task 2.1
├── tagger.go          ← You'll create this
├── tagger_test.go     ← Tests
└── docker/
    └── builder.go     ← Task 2.2
```

### Tagger Implementation

```go
// pkg/builder/tagger.go

package builder

import (
    "context"
    "fmt"
    "regexp"
    "time"
    
    "github.com/your-org/kudev/pkg/hash"
)

const (
    // TagPrefix identifies kudev-generated image tags.
    TagPrefix = "kudev-"
    
    // TimestampFormat is the format for timestamps in tags.
    // Uses UTC, sortable format: YYYYMMDD-HHMMSS
    TimestampFormat = "20060102-150405"
)

// tagPattern validates kudev tag format.
var tagPattern = regexp.MustCompile(`^kudev-[a-f0-9]{8}(-\d{8}-\d{6})?$`)

// Tagger generates image tags based on source code hash.
type Tagger struct {
    calculator *hash.Calculator
}

// NewTagger creates a new tagger with the given hash calculator.
func NewTagger(calculator *hash.Calculator) *Tagger {
    return &Tagger{
        calculator: calculator,
    }
}

// GenerateTag creates an image tag based on source hash.
// If forceTimestamp is true, appends UTC timestamp to force rebuild.
func (t *Tagger) GenerateTag(ctx context.Context, forceTimestamp bool) (string, error) {
    // Calculate source hash
    sourceHash, err := t.calculator.Calculate(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to calculate source hash: %w", err)
    }
    
    // Build tag
    tag := TagPrefix + sourceHash
    
    // Add timestamp if forced
    if forceTimestamp {
        timestamp := time.Now().UTC().Format(TimestampFormat)
        tag = fmt.Sprintf("%s-%s", tag, timestamp)
    }
    
    return tag, nil
}

// GetHash returns just the hash portion without generating full tag.
// Useful for cache checking before building.
func (t *Tagger) GetHash(ctx context.Context) (string, error) {
    return t.calculator.Calculate(ctx)
}

// IsKudevTag checks if a tag was generated by kudev.
func IsKudevTag(tag string) bool {
    return tagPattern.MatchString(tag)
}

// ParseTag extracts the hash from a kudev tag.
// Returns empty string if not a valid kudev tag.
func ParseTag(tag string) (hash string, hasTimestamp bool) {
    if !IsKudevTag(tag) {
        return "", false
    }
    
    // Remove prefix
    remainder := tag[len(TagPrefix):]
    
    // Check for timestamp
    if len(remainder) > 8 {
        return remainder[:8], true
    }
    
    return remainder, false
}

// TagInfo contains parsed information from a kudev tag.
type TagInfo struct {
    // Hash is the 8-character source hash.
    Hash string
    
    // HasTimestamp indicates if timestamp suffix was present.
    HasTimestamp bool
    
    // Timestamp is the parsed timestamp (if present).
    Timestamp time.Time
}

// ParseTagInfo extracts detailed information from a kudev tag.
func ParseTagInfo(tag string) (*TagInfo, error) {
    if !IsKudevTag(tag) {
        return nil, fmt.Errorf("not a kudev tag: %s", tag)
    }
    
    // Remove prefix
    remainder := tag[len(TagPrefix):]
    
    info := &TagInfo{
        Hash: remainder[:8],
    }
    
    // Check for timestamp
    if len(remainder) > 8 {
        info.HasTimestamp = true
        timestampStr := remainder[9:] // Skip the hyphen
        
        ts, err := time.Parse(TimestampFormat, timestampStr)
        if err != nil {
            return nil, fmt.Errorf("invalid timestamp in tag: %w", err)
        }
        info.Timestamp = ts
    }
    
    return info, nil
}

// CompareHashes checks if two tags have the same source hash.
// Useful for determining if rebuild is needed.
func CompareHashes(tag1, tag2 string) bool {
    hash1, _ := ParseTag(tag1)
    hash2, _ := ParseTag(tag2)
    
    if hash1 == "" || hash2 == "" {
        return false
    }
    
    return hash1 == hash2
}
```

---

## Key Implementation Details

### 1. Tag Format

Standard format is `kudev-{8-char-hash}`:
```go
tag := TagPrefix + sourceHash  // "kudev-a1b2c3d4"
```

With timestamp for forced rebuilds:
```go
timestamp := time.Now().UTC().Format(TimestampFormat)
tag = fmt.Sprintf("%s-%s", tag, timestamp)  // "kudev-a1b2c3d4-20250209-143025"
```

### 2. UTC Time

Always use UTC for consistency:
```go
time.Now().UTC().Format(TimestampFormat)
```

**Why UTC?**
- Consistent across timezones
- Sortable chronologically
- No daylight saving issues

### 3. Tag Validation

Use regex for validation:
```go
var tagPattern = regexp.MustCompile(`^kudev-[a-f0-9]{8}(-\d{8}-\d{6})?$`)

func IsKudevTag(tag string) bool {
    return tagPattern.MatchString(tag)
}
```

### 4. Hash Comparison

Compare just the hash portion (ignoring timestamps):
```go
func CompareHashes(tag1, tag2 string) bool {
    hash1, _ := ParseTag(tag1)
    hash2, _ := ParseTag(tag2)
    return hash1 == hash2
}
```

**Use case**: Check if rebuild needed
```go
currentHash, _ := tagger.GetHash(ctx)
existingHash, _ := ParseTag(existingImageTag)

if currentHash == existingHash {
    // No changes, skip rebuild
}
```

---

## Testing the Tagger

### Unit Tests

```go
// pkg/builder/tagger_test.go

package builder

import (
    "context"
    "os"
    "path/filepath"
    "strings"
    "testing"
    "time"
    
    "github.com/your-org/kudev/pkg/hash"
)

func TestGenerateTag(t *testing.T) {
    // Create temp directory with files
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    
    calc := hash.NewCalculator(tmpDir, nil)
    tagger := NewTagger(calc)
    ctx := context.Background()
    
    // Generate tag without timestamp
    tag, err := tagger.GenerateTag(ctx, false)
    if err != nil {
        t.Fatalf("GenerateTag failed: %v", err)
    }
    
    // Check format
    if !strings.HasPrefix(tag, TagPrefix) {
        t.Errorf("tag should start with %q, got %q", TagPrefix, tag)
    }
    
    // Should be exactly prefix + 8 chars
    expectedLen := len(TagPrefix) + 8
    if len(tag) != expectedLen {
        t.Errorf("tag length = %d, want %d", len(tag), expectedLen)
    }
}

func TestGenerateTag_WithTimestamp(t *testing.T) {
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    
    calc := hash.NewCalculator(tmpDir, nil)
    tagger := NewTagger(calc)
    ctx := context.Background()
    
    tag, err := tagger.GenerateTag(ctx, true)
    if err != nil {
        t.Fatalf("GenerateTag failed: %v", err)
    }
    
    // Should have timestamp suffix
    // Format: kudev-a1b2c3d4-20250209-143025
    expectedLen := len(TagPrefix) + 8 + 1 + 15 // prefix + hash + hyphen + timestamp
    if len(tag) != expectedLen {
        t.Errorf("tag length = %d, want %d (tag: %s)", len(tag), expectedLen, tag)
    }
}

func TestGenerateTag_Deterministic(t *testing.T) {
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    
    calc := hash.NewCalculator(tmpDir, nil)
    tagger := NewTagger(calc)
    ctx := context.Background()
    
    tag1, _ := tagger.GenerateTag(ctx, false)
    tag2, _ := tagger.GenerateTag(ctx, false)
    
    if tag1 != tag2 {
        t.Errorf("tags should be identical: %s != %s", tag1, tag2)
    }
}

func TestGenerateTag_ChangesWithContent(t *testing.T) {
    tmpDir := t.TempDir()
    mainFile := filepath.Join(tmpDir, "main.go")
    os.WriteFile(mainFile, []byte("package main"), 0644)
    
    calc := hash.NewCalculator(tmpDir, nil)
    tagger := NewTagger(calc)
    ctx := context.Background()
    
    tag1, _ := tagger.GenerateTag(ctx, false)
    
    // Modify file
    os.WriteFile(mainFile, []byte("package main\n// modified"), 0644)
    
    // Need new calculator for changed content
    calc2 := hash.NewCalculator(tmpDir, nil)
    tagger2 := NewTagger(calc2)
    tag2, _ := tagger2.GenerateTag(ctx, false)
    
    if tag1 == tag2 {
        t.Errorf("tags should differ after content change: %s == %s", tag1, tag2)
    }
}

func TestIsKudevTag(t *testing.T) {
    tests := []struct {
        tag      string
        expected bool
    }{
        {"kudev-a1b2c3d4", true},
        {"kudev-12345678", true},
        {"kudev-abcdef00", true},
        {"kudev-a1b2c3d4-20250209-143025", true},
        {"latest", false},
        {"v1.0.0", false},
        {"kudev-", false},
        {"kudev-abc", false},           // Too short
        {"kudev-abcdefghi", false},     // Too long (without timestamp)
        {"kudev-ABCD1234", false},      // Uppercase
        {"kudev-a1b2c3d4-invalid", false},
        {"", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.tag, func(t *testing.T) {
            result := IsKudevTag(tt.tag)
            if result != tt.expected {
                t.Errorf("IsKudevTag(%q) = %v, want %v", tt.tag, result, tt.expected)
            }
        })
    }
}

func TestParseTag(t *testing.T) {
    tests := []struct {
        tag          string
        wantHash     string
        wantHasTS    bool
    }{
        {"kudev-a1b2c3d4", "a1b2c3d4", false},
        {"kudev-12345678", "12345678", false},
        {"kudev-a1b2c3d4-20250209-143025", "a1b2c3d4", true},
        {"latest", "", false},
        {"", "", false},
    }
    
    for _, tt := range tests {
        t.Run(tt.tag, func(t *testing.T) {
            hash, hasTS := ParseTag(tt.tag)
            if hash != tt.wantHash {
                t.Errorf("ParseTag(%q) hash = %q, want %q", tt.tag, hash, tt.wantHash)
            }
            if hasTS != tt.wantHasTS {
                t.Errorf("ParseTag(%q) hasTimestamp = %v, want %v", tt.tag, hasTS, tt.wantHasTS)
            }
        })
    }
}

func TestParseTagInfo(t *testing.T) {
    // Test basic tag
    info, err := ParseTagInfo("kudev-a1b2c3d4")
    if err != nil {
        t.Fatalf("ParseTagInfo failed: %v", err)
    }
    if info.Hash != "a1b2c3d4" {
        t.Errorf("Hash = %q, want %q", info.Hash, "a1b2c3d4")
    }
    if info.HasTimestamp {
        t.Error("HasTimestamp should be false")
    }
    
    // Test tag with timestamp
    info, err = ParseTagInfo("kudev-a1b2c3d4-20250209-143025")
    if err != nil {
        t.Fatalf("ParseTagInfo failed: %v", err)
    }
    if info.Hash != "a1b2c3d4" {
        t.Errorf("Hash = %q, want %q", info.Hash, "a1b2c3d4")
    }
    if !info.HasTimestamp {
        t.Error("HasTimestamp should be true")
    }
    
    expectedTime := time.Date(2025, 2, 9, 14, 30, 25, 0, time.UTC)
    if !info.Timestamp.Equal(expectedTime) {
        t.Errorf("Timestamp = %v, want %v", info.Timestamp, expectedTime)
    }
}

func TestCompareHashes(t *testing.T) {
    tests := []struct {
        tag1     string
        tag2     string
        expected bool
    }{
        {"kudev-a1b2c3d4", "kudev-a1b2c3d4", true},
        {"kudev-a1b2c3d4", "kudev-a1b2c3d4-20250209-143025", true},
        {"kudev-a1b2c3d4-20250209-143025", "kudev-a1b2c3d4-20250210-100000", true},
        {"kudev-a1b2c3d4", "kudev-e5f6g7h8", false},
        {"kudev-a1b2c3d4", "latest", false},
        {"latest", "v1.0.0", false},
    }
    
    for _, tt := range tests {
        name := tt.tag1 + "_vs_" + tt.tag2
        t.Run(name, func(t *testing.T) {
            result := CompareHashes(tt.tag1, tt.tag2)
            if result != tt.expected {
                t.Errorf("CompareHashes(%q, %q) = %v, want %v", 
                    tt.tag1, tt.tag2, result, tt.expected)
            }
        })
    }
}

func TestGetHash(t *testing.T) {
    tmpDir := t.TempDir()
    os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0644)
    
    calc := hash.NewCalculator(tmpDir, nil)
    tagger := NewTagger(calc)
    ctx := context.Background()
    
    hash, err := tagger.GetHash(ctx)
    if err != nil {
        t.Fatalf("GetHash failed: %v", err)
    }
    
    if len(hash) != 8 {
        t.Errorf("hash length = %d, want 8", len(hash))
    }
    
    // Verify GetHash matches tag hash
    tag, _ := tagger.GenerateTag(ctx, false)
    tagHash, _ := ParseTag(tag)
    
    if hash != tagHash {
        t.Errorf("GetHash() = %q, tag hash = %q", hash, tagHash)
    }
}
```

---

## Usage Examples

### Basic Tag Generation

```go
// Create calculator for source directory
calc := hash.NewCalculator("/path/to/project", nil)
tagger := builder.NewTagger(calc)

// Generate tag
tag, err := tagger.GenerateTag(ctx, false)
// tag = "kudev-a1b2c3d4"

// Use with builder
opts := builder.BuildOptions{
    ImageName: "myapp",
    ImageTag:  tag,
    // ...
}
```

### Forced Rebuild with Timestamp

```go
// Force new tag (for debugging, cache issues)
tag, err := tagger.GenerateTag(ctx, true)
// tag = "kudev-a1b2c3d4-20250209-143025"
```

### Cache Check Before Build

```go
// Get current source hash
currentHash, _ := tagger.GetHash(ctx)

// Get hash from existing image
existingTag := getExistingImageTag() // "kudev-a1b2c3d4"
existingHash, _ := builder.ParseTag(existingTag)

// Check if rebuild needed
if currentHash == existingHash {
    fmt.Println("Source unchanged, skipping build")
    return
}

// Source changed, rebuild
tag, _ := tagger.GenerateTag(ctx, false)
// Build with new tag...
```

### Tag Cleanup

```go
// List all kudev-tagged images for cleanup
images := listDockerImages() // ["myapp:kudev-a1b2c3d4", "myapp:latest", ...]

for _, img := range images {
    tag := strings.Split(img, ":")[1]
    if builder.IsKudevTag(tag) {
        // This is a kudev-generated image
        // Safe to cleanup if old
    }
}
```

---

## How Tagger Connects to Other Tasks

```
Task 2.4 (Tagger) ← You are here
    │
    ├─► Uses: Task 2.3 (Hash Calculator)
    │   - Gets source hash for tag generation
    │
    └─► Used by: Task 2.2 (Docker Builder)
        - Provides ImageTag for BuildOptions
```

### Integration Example

```go
// Full build pipeline
func BuildImage(ctx context.Context, projectRoot string, imageName string) (*builder.ImageRef, error) {
    // 1. Create hash calculator
    calc := hash.NewCalculator(projectRoot, nil)
    
    // 2. Create tagger
    tagger := builder.NewTagger(calc)
    
    // 3. Generate tag
    tag, err := tagger.GenerateTag(ctx, false)
    if err != nil {
        return nil, fmt.Errorf("failed to generate tag: %w", err)
    }
    
    // 4. Create builder
    db := docker.NewDockerBuilder(logger)
    
    // 5. Build image
    opts := builder.BuildOptions{
        SourceDir:      projectRoot,
        DockerfilePath: "./Dockerfile",
        ImageName:      imageName,
        ImageTag:       tag,
    }
    
    return db.Build(ctx, opts)
}
```

---

## Checklist for Task 2.4

- [ ] Create `pkg/builder/tagger.go`
- [ ] Define `TagPrefix` and `TimestampFormat` constants
- [ ] Define `tagPattern` regex
- [ ] Implement `Tagger` struct
- [ ] Implement `NewTagger()` constructor
- [ ] Implement `GenerateTag()` method
- [ ] Implement `GetHash()` method
- [ ] Implement `IsKudevTag()` function
- [ ] Implement `ParseTag()` function
- [ ] Implement `TagInfo` struct
- [ ] Implement `ParseTagInfo()` function
- [ ] Implement `CompareHashes()` function
- [ ] Create `pkg/builder/tagger_test.go`
- [ ] Write tests for `GenerateTag()`
- [ ] Write tests for determinism
- [ ] Write tests for `IsKudevTag()`
- [ ] Write tests for `ParseTag()`
- [ ] Write tests for `CompareHashes()`
- [ ] Run `go fmt ./pkg/builder`
- [ ] Verify compilation: `go build ./pkg/builder`
- [ ] Run tests: `go test ./pkg/builder -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Using local time
```go
// Wrong - varies by timezone
timestamp := time.Now().Format(TimestampFormat)

// Right - consistent UTC
timestamp := time.Now().UTC().Format(TimestampFormat)
```

❌ **Mistake 2**: Weak tag validation
```go
// Wrong - accepts invalid tags
func IsKudevTag(tag string) bool {
    return strings.HasPrefix(tag, "kudev-")
}

// Right - strict format validation
var tagPattern = regexp.MustCompile(`^kudev-[a-f0-9]{8}(-\d{8}-\d{6})?$`)
func IsKudevTag(tag string) bool {
    return tagPattern.MatchString(tag)
}
```

❌ **Mistake 3**: Ignoring errors from hash calculation
```go
// Wrong - silent failures
tag := TagPrefix + calculator.Calculate(ctx)  // Ignores error!

// Right - propagate errors
hash, err := calculator.Calculate(ctx)
if err != nil {
    return "", fmt.Errorf("failed to calculate hash: %w", err)
}
tag := TagPrefix + hash
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 2.5** → Implement Registry Loading
3. Registry loader will use ImageRef to load images to clusters

---

## References

- [Docker Image Tag Reference](https://docs.docker.com/engine/reference/commandline/tag/)
- [Go Time Package](https://pkg.go.dev/time)
- [Go Regexp Package](https://pkg.go.dev/regexp)
- [K8s Image Pull Policy](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)

