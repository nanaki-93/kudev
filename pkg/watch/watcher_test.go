// pkg/watch/watcher_test.go

package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nanaki-93/kudev/test/util"
)

func TestFSWatcher_DetectsFileChange(t *testing.T) {
	tmpDir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main"), 0644)

	// Create watcher
	watcher, err := NewFSWatcher(nil, &util.MockLogger{})
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

	watcher, _ := NewFSWatcher(nil, &util.MockLogger{})
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

	watcher, _ := NewFSWatcher(nil, &util.MockLogger{})
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
