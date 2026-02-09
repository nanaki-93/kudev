package builder

import (
	"context"
	"fmt"
	"strings"
)

type Builder interface {
	Build(ctx context.Context, opts BuildOptions) (*ImageRef, error)
	Name() string
}

type BuildOptions struct {
	SourceDir      string
	DockerfilePath string
	ImageName      string
	ImageTag       string
	BuildArgs      map[string]string
	Target         string
	NoCache        bool
}

type ImageRef struct {
	FullRef string
	ID      string
	Digest  string
}

func (r *ImageRef) String() string {
	return r.FullRef
}

type Factory func() (Builder, error)

func (o BuildOptions) Validate() error {
	var errors []string
	if o.SourceDir == "" {
		errors = append(errors, "SourceDir is required")
	}

	if o.DockerfilePath == "" {
		errors = append(errors, "DockerfilePath is required")
	}

	if o.ImageName == "" {
		errors = append(errors, "ImageName is required")
	}

	if o.ImageTag == "" {
		errors = append(errors, "ImageTag is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid BuildOptions: %s", strings.Join(errors, ", "))
	}
	return nil
}
