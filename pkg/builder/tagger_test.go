package builder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nanaki-93/kudev/pkg/hash"
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
		{"kudev-abc", false},       // Too short
		{"kudev-abcdefghi", false}, // Too long (without timestamp)
		{"kudev-ABCD1234", false},  // Uppercase
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
		tag       string
		wantHash  string
		wantHasTS bool
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
