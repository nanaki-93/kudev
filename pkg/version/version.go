package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the kudev version (set at build time)
	// go build -ldflags="-X github.com/yourusername/kudev/pkg/version.Version=v0.1.0"
	Version = "v0.1.0-dev"

	// GitCommit is the git commit hash (set at build time)
	GitCommit = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = fmt.Sprintf("Go %s", runtime.Version())

	// OS is the operating system
	OS = runtime.GOOS

	// Arch is the CPU architecture
	Arch = runtime.GOARCH
)
