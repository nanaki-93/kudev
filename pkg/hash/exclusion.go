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
