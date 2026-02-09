package docker

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/nanaki-93/kudev/pkg/builder"
	"github.com/nanaki-93/kudev/pkg/logging"
)

type Builder struct {
	logger logging.LoggerInterface
}

func NewBuilder(logger logging.LoggerInterface) *Builder {
	return &Builder{logger: logger}
}

func (b *Builder) Name() string {
	return "docker"
}

func (b *Builder) Build(ctx context.Context, opts builder.BuildOptions) (*builder.ImageRef, error) {
	// Validate options first
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid build options: %w", err)
	}

	// 1. Verify Docker daemon is running
	if err := b.checkDockerDaemon(ctx); err != nil {
		return nil, err
	}

	b.logger.Info("starting docker build",
		"image", opts.ImageName,
		"tag", opts.ImageTag,
		"dockerfile", opts.DockerfilePath,
	)

	// 2. Build docker command arguments
	args := b.buildCommandArgs(opts)

	// 3. Create command with context for cancellation
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = opts.SourceDir // Set working directory to source

	// 4. Get stdout and stderr pipes for streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 5. Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start docker build: %w", err)
	}

	// 6. Stream output in goroutines
	go b.streamOutput("stdout", stdout)
	go b.streamOutput("stderr", stderr)

	// 7. Wait for completion
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("docker build failed: %w", err)
	}

	b.logger.Info("docker build completed successfully")

	// 8. Get image ID
	fullRef := fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag)
	imageID, err := b.getImageID(ctx, fullRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get image ID: %w", err)
	}

	return &builder.ImageRef{
		FullRef: fullRef,
		ID:      imageID,
	}, nil
}

// checkDockerDaemon verifies the Docker daemon is running and accessible.
func (b *Builder) checkDockerDaemon(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"docker daemon is not running or not accessible\n\n"+
				"Troubleshooting:\n"+
				"  1. Ensure Docker Desktop is running\n"+
				"  2. Or start Docker daemon: sudo systemctl start docker\n"+
				"  3. Verify with: docker version\n\n"+
				"Error: %w\nOutput: %s", err, string(output),
		)
	}

	b.logger.Debug("docker daemon available", "version", strings.TrimSpace(string(output)))
	return nil
}

// buildCommandArgs constructs the docker build command arguments.
func (b *Builder) buildCommandArgs(opts builder.BuildOptions) []string {
	args := []string{"build"}

	// Add tag
	args = append(args, "-t", fmt.Sprintf("%s:%s", opts.ImageName, opts.ImageTag))

	// Add Dockerfile path
	args = append(args, "-f", opts.DockerfilePath)

	// Add build args
	for key, val := range opts.BuildArgs {
		args = append(args, "--build-arg", fmt.Sprintf("%s=%s", key, val))
	}

	// Add target if specified
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}

	// Add no-cache if specified
	if opts.NoCache {
		args = append(args, "--no-cache")
	}

	// Add build context (current directory since we set cmd.Dir)
	args = append(args, ".")

	return args
}

// streamOutput reads from a reader and logs each line.
func (b *Builder) streamOutput(source string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			b.logger.Info(line, "source", source)
		}
	}

	if err := scanner.Err(); err != nil {
		b.logger.Error(err, "error reading output", "source", source)
	}
}

// getImageID retrieves the image ID using docker inspect.
func (b *Builder) getImageID(ctx context.Context, imageRef string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect",
		"--format={{.ID}}", imageRef)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect image %s: %w", imageRef, err)
	}

	imageID := strings.TrimSpace(string(output))
	b.logger.Debug("retrieved image ID", "image", imageRef, "id", imageID)

	return imageID, nil
}

// Ensure DockerBuilder implements builder.Builder
var _ builder.Builder = (*Builder)(nil)
