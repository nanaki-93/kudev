package commands

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/nanaki-93/kudev/pkg/config"
	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize kudev configuration",
	Long: `Initialize a new .kudev.yaml configuration file.

This command guides you through setup:
  - Project name (used as deployment name)
  - Dockerfile path
  - Kubernetes namespace
  - Container ports

The configuration is saved to .kudev.yaml in the current directory.

Examples:
  kudev init                  Interactive mode
  kudev init my-app           Create config for 'my-app'
  kudev init my-app --namespace production
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := logging.Get()

		var appName string
		if len(args) > 0 {
			appName = args[0]
		}

		// Start interactive setup
		cfg, err := interactiveSetup(appName)
		if err != nil {
			return err
		}

		// Validate before saving
		if err := cfg.Validate(cmd.Context()); err != nil {
			return err
		}

		// Save to file
		configPath := ".kudev.yaml"
		loader := config.NewFileConfigLoader("", "", "")

		if err := loader.Save(cmd.Context(), cfg, configPath); err != nil {
			return fmt.Errorf("failed to save configuration: %w", err)
		}

		logger.Info(
			"configuration file created successfully",
			"path", configPath,
		)

		fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
		fmt.Printf("\nNext steps:\n")
		fmt.Printf("  1. Review the configuration: cat %s\n", configPath)
		fmt.Printf("  2. Validate the configuration: kudev validate\n")
		fmt.Printf("  3. Deploy to Kubernetes: kudev up\n")

		return nil
	},
}

// interactiveSetup guides user through configuration creation.
func interactiveSetup(appName string) (*config.DeploymentConfig, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nKudev Configuration Setup")
	fmt.Println("========================================")

	// App name
	if appName == "" {
		fmt.Print("\nProject name (e.g., my-app): ")
		name, _ := reader.ReadString('\n')
		appName = strings.TrimSpace(name)
	}

	if appName == "" {
		return nil, fmt.Errorf("project name is required")
	}

	// Dockerfile path
	fmt.Print("Dockerfile path [./Dockerfile]: ")
	dockerfilePath, _ := reader.ReadString('\n')
	dockerfilePath = strings.TrimSpace(dockerfilePath)
	if dockerfilePath == "" {
		dockerfilePath = "./Dockerfile"
	}

	// Namespace
	fmt.Print("Kubernetes namespace [default]: ")
	namespace, _ := reader.ReadString('\n')
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		namespace = "default"
	}

	// Replicas
	fmt.Print("Number of replicas [1]: ")
	replicasStr, _ := reader.ReadString('\n')
	replicasStr = strings.TrimSpace(replicasStr)
	replicas := int32(1)
	if replicasStr != "" {
		if r, err := strconv.ParseInt(replicasStr, 10, 32); err == nil {
			replicas = int32(r)
		}
	}

	// Service port
	fmt.Print("Container port [8080]: ")
	servicePortStr, _ := reader.ReadString('\n')
	servicePortStr = strings.TrimSpace(servicePortStr)
	servicePort := int32(8080)
	if servicePortStr != "" {
		if p, err := strconv.ParseInt(servicePortStr, 10, 32); err == nil {
			servicePort = int32(p)
		}
	}

	// Local port
	fmt.Print("Local port for forwarding [8080]: ")
	localPortStr, _ := reader.ReadString('\n')
	localPortStr = strings.TrimSpace(localPortStr)
	localPort := int32(8080)
	if localPortStr != "" {
		if p, err := strconv.ParseInt(localPortStr, 10, 32); err == nil {
			localPort = int32(p)
		}
	}

	// Build config
	cfg := &config.DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: config.MetadataConfig{
			Name: appName,
		},
		Spec: config.SpecConfig{
			ImageName:      appName,
			DockerfilePath: dockerfilePath,
			Namespace:      namespace,
			Replicas:       replicas,
			LocalPort:      localPort,
			ServicePort:    servicePort,
		},
	}

	config.ApplyDefaults(cfg)

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("Configuration Summary:")
	fmt.Printf("  Project: %s\n", cfg.Metadata.Name)
	fmt.Printf("  Dockerfile: %s\n", cfg.Spec.DockerfilePath)
	fmt.Printf("  Namespace: %s\n", cfg.Spec.Namespace)
	fmt.Printf("  Replicas: %d\n", cfg.Spec.Replicas)
	fmt.Printf("  Service Port: %d\n", cfg.Spec.ServicePort)
	fmt.Printf("  Local Port: %d\n", cfg.Spec.LocalPort)
	fmt.Println(strings.Repeat("=", 40))

	return cfg, nil
}
