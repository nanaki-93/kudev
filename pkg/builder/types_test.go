package builder

import (
	"testing"
)

func TestBuildOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    BuildOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
			},
			wantErr: false,
		},
		{
			name: "missing SourceDir",
			opts: BuildOptions{
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
			},
			wantErr: true,
		},
		{
			name: "missing DockerfilePath",
			opts: BuildOptions{
				SourceDir: "/project",
				ImageName: "myapp",
				ImageTag:  "kudev-abc123",
			},
			wantErr: true,
		},
		{
			name: "missing ImageName",
			opts: BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageTag:       "kudev-abc123",
			},
			wantErr: true,
		},
		{
			name: "missing ImageTag",
			opts: BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
			},
			wantErr: true,
		},
		{
			name:    "all fields missing",
			opts:    BuildOptions{},
			wantErr: true,
		},
		{
			name: "with optional fields",
			opts: BuildOptions{
				SourceDir:      "/project",
				DockerfilePath: "./Dockerfile",
				ImageName:      "myapp",
				ImageTag:       "kudev-abc123",
				BuildArgs:      map[string]string{"VERSION": "1.0"},
				Target:         "runtime",
				NoCache:        true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestImageRefString(t *testing.T) {
	ref := ImageRef{
		FullRef: "myapp:kudev-abc123",
		ID:      "sha256:abc123",
	}

	if ref.String() != "myapp:kudev-abc123" {
		t.Errorf("String() = %v, want %v", ref.String(), "myapp:kudev-abc123")
	}
}
