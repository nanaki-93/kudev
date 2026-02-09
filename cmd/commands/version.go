package commands

import (
	"fmt"

	"github.com/nanaki-93/kudev/pkg/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print the version of kudev and related components.

Shows:
  - kudev version
  - Go version
  - OS/Architecture
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kudev version " + version.Version)
		fmt.Println("Built with " + version.GoVersion)
		fmt.Printf("OS/Arch: %s/%s\n", version.OS, version.Arch)
	},
}
