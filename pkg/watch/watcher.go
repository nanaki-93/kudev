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

	"github.com/nanaki-93/kudev/pkg/logging"
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
	logger     logging.LoggerInterface
}

// NewFSWatcher creates a new file system watcher.
func NewFSWatcher(exclusions []string, logger logging.LoggerInterface) (*FSWatcher, error) {
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
			w.logger.Error(err, "watcher error")
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
