package benchmark

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nanaki-93/kudev/pkg/hash"
)

func BenchmarkCalculate(b *testing.B) {
	// Create test directory with realistic content
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		content := fmt.Sprintf("package p%d\n// content %d", i, i)
		filename := fmt.Sprintf("file%d.go", i)
		os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
	}

	calc := hash.NewCalculator(tmpDir, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calc.Calculate(ctx)
	}
}
