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
