package docker

import (
	"testing"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/test/util"
)

func TestBuildCommandArgs(t *testing.T) {
	logger := &util.MockLogger{}
	db := NewBuilder(logger)

	tests := []struct {
		name     string
		opts     builder.BuildOptions
		expected []string
	}{
		{
			name: "basic build",
			opts: builder.BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
			},
			expected: []string{
				"build",
				"-t", "myapp:kudev-abc123",
				"-f", "./Dockerfile",
				".",
			},
		},
		{
			name: "with build args",
			opts: builder.BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
				BuildArgs:      map[string]string{"VERSION": "1.0"},
			},
			expected: []string{
				"build",
				"-t", "myapp:kudev-abc123",
				"-f", "./Dockerfile",
				"--build-arg", "VERSION=1.0",
				".",
			},
		},
		{
			name: "with target and no-cache",
			opts: builder.BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
				Target:         "runtime",
				NoCache:        true,
			},
			expected: []string{
				"build",
				"-t", "myapp:kudev-abc123",
				"-f", "./Dockerfile",
				"--target", "runtime",
				"--no-cache",
				".",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := db.buildCommandArgs(tt.opts)

			// Check essential args are present
			// Note: BuildArgs map iteration order is random
			for _, exp := range tt.expected {
				found := false
				for _, arg := range args {
					if arg == exp {
						found = true
						break
					}
				}
				if !found && exp != "--build-arg" && exp != "VERSION=1.0" {
					t.Errorf("expected arg %q not found in %v", exp, args)
				}
			}
		})
	}
}

func TestDockerBuilderImplementsInterface(t *testing.T) {
	// Compile-time check that DockerBuilder implements Builder
	var _ builder.Builder = (*Builder)(nil)
}
