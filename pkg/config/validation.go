package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func (c *DeploymentConfig) Validate(ctx context.Context) error {
	var errs ValidationError

	if c == nil {
		return fmt.Errorf("config is nil")
	}
	if c.APIVersion == "" {
		errs.Add("apiVersion is required (should be: kudev.io/v1alpha1)")
	} else if c.APIVersion != "kudev.io/v1alpha1" {
		errs.Add(fmt.Sprintf("apiVersion must be 'kudev.io/v1alpha1', got '%s'", c.APIVersion))
	}

	if c.Kind == "" {
		errs.Add("kind is required (should be: DeploymentConfig)")
	} else if c.Kind != "DeploymentConfig" {
		errs.Add(fmt.Sprintf("kind must be 'DeploymentConfig', got '%s'", c.Kind))
	}

	errs.Merge(c.validateMetadata())
	errs.Merge(c.validateSpec(ctx))

	if errs.HasErrors() {
		return &errs
	}
	return nil

}

func (c *DeploymentConfig) validateMetadata() ValidationError {
	var errs ValidationError

	if c.Metadata.Name == "" {
		errs.Add("metadata.name is required and cannot be empty")
		errs.AddExample("metadata:\n  name: my-app")
		return errs
	}
	if err := validateDNSName(c.Metadata.Name); err != nil {
		errs.Add(fmt.Sprintf("metadata.name: %v", err))
		errs.AddExample("metadata:\n  name: my-app  # lowercase, hyphens, alphanumeric")
	}

	return errs
}

// ctx is passed for future cancellation support (e.g., if file system I/O)

func (c *DeploymentConfig) validateSpec(ctx context.Context) ValidationError {
	var errs ValidationError

	spec := c.Spec
	if spec.ImageName == "" {
		errs.Add("spec.imageName is required")
		errs.AddExample("spec:\n  imageName: my-app")
		// Don't return early - validate other fields too
	}

	if spec.DockerfilePath == "" {
		errs.Add("spec.dockerfilePath is required")
		errs.AddExample("spec:\n  dockerfilePath: ./Dockerfile")
	} else {
		// Validate the dockerfile path (only if not empty)
		if err := validateDockerfilePath(spec.DockerfilePath); err != nil {
			errs.Add(fmt.Sprintf("spec.dockerfilePath: %v", err))
		}
	}

	if spec.Namespace == "" {
		errs.Add("spec.namespace is required")
		errs.AddExample("spec:\n  namespace: default")
		// Don't return early - let user see all problems
	} else {
		if err := validateDNSName(spec.Namespace); err != nil {
			errs.Add(fmt.Sprintf("spec.namespace: %v", err))
			errs.AddExample("spec:\n  namespace: default  # or: dev, prod-staging")
		}
	}

	// === Numeric Constraints ===

	if spec.Replicas < 1 {
		errs.Add(fmt.Sprintf(
			"spec.replicas must be at least 1, got %d",
			spec.Replicas,
		))
		errs.AddExample("spec:\n  replicas: 1")
	}

	if spec.Replicas > 100 {
		// Warning, not error (but we're only doing errors for now)
		// Phase 4 can add warnings system
	}

	// === Port Validation ===

	if err := validatePort("spec.localPort", spec.LocalPort); err != nil {
		errs.Add(err.Error())
		errs.AddExample("spec:\n  localPort: 8080  # 1-65535")
	}

	if err := validatePort("spec.servicePort", spec.ServicePort); err != nil {
		errs.Add(err.Error())
		errs.AddExample("spec:\n  servicePort: 8080  # 1-65535")
	}

	// === Environment Variables ===

	if err := validateEnv(spec.Env); err != nil {
		errs.Merge(*err)
	}

	// === Optional Fields ===

	if spec.ImageName != "" {
		if err := validateImageName(spec.ImageName); err != nil {
			errs.Add(fmt.Sprintf("spec.imageName: %v", err))
			errs.AddExample("spec:\n  imageName: my-app  # lowercase, hyphens")
		}
	}

	if spec.KubeContext != "" {
		// Note: Actual context validation happens in Task 1.4
		// Here we just check format
		if err := validateKubeContextName(spec.KubeContext); err != nil {
			errs.Add(fmt.Sprintf("spec.kubeContext: %v", err))
		}
	}

	if len(spec.BuildContextExclusions) > 0 {
		if err := validateBuildContextExclusions(spec.BuildContextExclusions); err != nil {
			errs.Merge(*err)
		}
	}

	return errs
}

func validateDNSName(name string) error {

	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf(
			"must be DNS-1123 compliant (lowercase alphanumeric and hyphens only, cannot start/end with hyphen). Got '%q'", name,
		)
	}
	if len(name) < 3 {
		return fmt.Errorf("must be at least 3 characters long, got '%q'", len(name))
	}

	if len(name) > 63 {
		return fmt.Errorf("must be at most 63 characters long, got '%q'", len(name))
	}
	return nil

}

func validateDockerfilePath(path string) error {
	if path == "" {
		return fmt.Errorf("dockerfilePath cannot be empty")
	}

	path = filepath.Clean(path)

	if strings.HasSuffix(path, ".git") {
		return fmt.Errorf("dockerfile path cannot be .git")
	}

	if strings.HasSuffix(path, ".kudev.yaml") {
		return fmt.Errorf("dockerfile path cannot be .kudev.yaml")
	}

	base := filepath.Base(path)
	if !strings.HasPrefix(base, "Dockerfile") && !strings.HasPrefix(base, "dockerfile") {
		if !strings.Contains(base, "docker") && !strings.Contains(base, "Docker") {
			return fmt.Errorf(
				"expected filename to contain 'Dockerfile', got '%s' "+
					"(examples: Dockerfile, Dockerfile.dev, docker/Dockerfile)", base)
		}
	}

	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file '%s' does not exist", path)
			}
			return fmt.Errorf("cannot access file '%s': %v", path, err)
		}
	}
	// If relative path, we can't validate without knowing the project root
	// That happens in loader.go with full context

	return nil
}

func validatePort(fieldName string, port int32) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535, got %d", fieldName, port)
	}
	if port < 1024 {
		//just a warning, but we want to encourage users to use non-privileged ports
		fmt.Printf("Warning: %s is set to %d, which is a privileged port. Consider using a port above 1024 to avoid permission issues.\n", fieldName, port)
	}
	return nil
}

func validateImageName(name string) error {
	pattern := `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf("must be lowercase alphanumeric and hypens only. Got: %q", name)
	}
	if len(name) > 128 {
		return fmt.Errorf("must be at most 128 characters long, got '%d'", len(name))
	}
	return nil
}

func validateEnv(vars []EnvVar) *ValidationError {
	var errs ValidationError
	if len(vars) == 0 {
		return &errs // empty env vars are valid
	}

	seenNames := make(map[string]bool)
	for i, v := range vars {
		if v.Name == "" {
			errs.Add(fmt.Sprintf("env[%d].name is required", i))
			errs.AddExample("env:\n- name: LOG_LEVEL\n  value: debug")
			continue
		}
		if err := validateEnvVarName(v.Name); err != nil {
			errs.Add(fmt.Sprintf("env[%d].name %q: %v", i, v.Name, err))
		}

		if seenNames[v.Name] {
			errs.Add(fmt.Sprintf("env[%d].name '%q' is not unique (first occurence: env[?].name %q)", i, v.Name, v.Name))
		}
		seenNames[v.Name] = true
	}
	return &errs
}

func validateEnvVarName(name string) error {
	if name == "" {
		return errors.New("name is required")
	}
	pattern := `^[A-Z_][A-Z0-9_]*$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf("invalid format. Environment variable names must be UPPERCASE_WITH_UNDERSCORES. "+
			"Got: %q (examples: LOG_LEVEL, DATABASE_URL, DEBUG_MODE)",
			name)
	}
	return nil
}

func validateKubeContextName(name string) error {
	if name == "" {
		return errors.New("kubeContext cannot be empty if specified")
	}
	pattern := `^[a-zA-Z0-9._\-/]+$`
	if !regexp.MustCompile(pattern).MatchString(name) {
		return fmt.Errorf("invalid context name format: %q (examples: docker-desktop, kind-dev, my.context)",
			name)
	}
	return nil
}

func validateBuildContextExclusions(exclusions []string) *ValidationError {
	var errs ValidationError

	for i, exc := range exclusions {
		if exc == "" {
			errs.Add(fmt.Sprintf("buildContextExclusions[%d] cannot be empty", i))
			continue
		}

		if strings.HasPrefix(exc, "/") {
			errs.Add(fmt.Sprintf("buildContextExclusions[%d] should be relative path, not absolute: %q", i, exc))
		}

		if strings.Contains(exc, "\\") {
			errs.Add(fmt.Sprintf("buildContextExclusions[%d] should use forward slashes, not backslashes: %q (use '%s')",
				i, exc, strings.ReplaceAll(exc, "\\", "/")))
		}
	}
	return &errs
}

func (c *DeploymentConfig) ValidateWithContext(projectRoot string) error {
	if err := c.Validate(context.Background()); err != nil {
		return err
	}
	var errs ValidationError

	dockerfilePath := c.Spec.DockerfilePath
	if !filepath.IsAbs(dockerfilePath) {
		dockerfilePath = filepath.Join(projectRoot, dockerfilePath)
	}

	if _, err := os.Stat(dockerfilePath); err != nil {
		errs.Add(fmt.Sprintf("spec.dockerfilePath '%q' does not exist at %s", c.Spec.DockerfilePath, dockerfilePath))
	}

	if errs.HasErrors() {
		return &errs
	}

	return nil
}
