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
