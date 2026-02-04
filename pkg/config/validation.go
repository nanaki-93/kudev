package config

import (
	"fmt"
	"regexp"
)

var dnsRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func isValidDNS(name string) bool {
	if len(name) == 0 || len(name) > 253 {
		return false
	}
	return dnsRegex.MatchString(name)
}

func isValidPort(port int32) bool {
	return port > 0 && port <= 65535
}

func (c *DeploymentConfig) Validate() error {
	ve := &ValidationErrors{}

	if c.APIVersion == "" {
		ve.Add("apiVersion", "is required")
	} else if c.APIVersion != "kudev.io/v1alpha1" {
		ve.Add("apiVersion", "unsupported version")
	}

	if c.Kind == "" {
		ve.Add("kind", "is required")
	} else if c.Kind != "DeploymentConfig" {
		ve.Add("kind", "unsupported kind")
	}

	c.ValidateMetadata(ve)
	c.ValidateSpec(ve)
	c.ValidateEnv(ve)

	if ve.HasErrors() {
		return ve
	}
	return nil
}

func (c *DeploymentConfig) ValidateMetadata(ve *ValidationErrors) {
	if c.Metadata.Name == "" {
		ve.Add("metadata.name", "is required")
		return
	}
	if !isValidDNS(c.Metadata.Name) {
		ve.Add("metadata.name", fmt.Sprintf(
			"must be DNS-1123 compliant (lowercase letters, numbers, hyphens; max 253 chars), got '%s'",
			c.Metadata.Name,
		))
	}
}

func (c *DeploymentConfig) ValidateSpec(ve *ValidationErrors) {
	if c.Spec.ImageName == "" {
		ve.Add("spec.imageName", "is required")
	}
	if c.Spec.DockerFilePath == "" {
		ve.Add("spec.dockerFilePath", "is required")
	}
	if c.Spec.Namespace == "" {
		ve.Add("spec.namespace", "is required")
	} else if !isValidDNS(c.Spec.Namespace) {
		ve.Add("spec.namespace", fmt.Sprintf(
			"must be DNS-1123 compliant (lowercase letters, numbers, hyphens; max 253 chars), got '%s'",
			c.Spec.Namespace,
		))
	}
	if c.Spec.LocalPort != 0 && !isValidPort(c.Spec.LocalPort) {
		ve.Add("spec.localPort", fmt.Sprintf("must be between 1 and 65535, got %d", c.Spec.LocalPort))

	}
	if c.Spec.ServicePort != 0 && !isValidPort(c.Spec.ServicePort) {
		ve.Add("spec.servicePort", fmt.Sprintf("must be between 1 and 65535, got %d", c.Spec.ServicePort))

	}
	if c.Spec.Replicas < 1 {
		ve.Add("spec.replicas", fmt.Sprintf("must be at least 1, got %d", c.Spec.Replicas))
	}

	if c.Spec.KubeContext != "" && !isValidDNS(c.Spec.KubeContext) {
		ve.Add("spec.kubeContext", fmt.Sprintf(
			"must be DNS-1123 compliant (lowercase letters, numbers, hyphens; max 253 chars), got '%s'",
			c.Spec.KubeContext,
		))
	}

}

func (c *DeploymentConfig) ValidateEnv(ve *ValidationErrors) {
	for i, envVar := range c.Spec.Env {
		if envVar.Name == "" {
			ve.Add(fmt.Sprintf("spec.env[%d].name", i), "is required")
		}
	}
}

func (c *DeploymentConfig) ValidateField(fieldPath string) error {

	switch fieldPath {
	case "metadata.name":
		if c.Metadata.Name == "" {
			return fmt.Errorf("name is required")
		}
		if !isValidDNS(c.Metadata.Name) {
			return fmt.Errorf("metadata.name: must be DNS-1123 compliant, got '%s'", c.Metadata.Name)
		}
	case "spec.imageName":
		if c.Spec.ImageName == "" {
			return fmt.Errorf("spec.imageName: is required")
		}

	case "spec.dockerfilePath":
		if c.Spec.DockerFilePath == "" {
			return fmt.Errorf("spec.dockerfilePath: is required")
		}

	case "spec.namespace":
		if c.Spec.Namespace == "" {
			return fmt.Errorf("spec.namespace: is required")
		}
		if !isValidDNS(c.Spec.Namespace) {
			return fmt.Errorf("spec.namespace: must be DNS-1123 compliant, got '%s'", c.Spec.Namespace)
		}

	case "spec.localPort":
		if c.Spec.LocalPort != 0 && !isValidPort(c.Spec.LocalPort) {
			return fmt.Errorf("spec.localPort: must be between 1 and 65535, got %d", c.Spec.LocalPort)
		}

	case "spec.servicePort":
		if c.Spec.ServicePort != 0 && !isValidPort(c.Spec.ServicePort) {
			return fmt.Errorf("spec.servicePort: must be between 1 and 65535, got %d", c.Spec.ServicePort)
		}

	case "spec.replicas":
		if c.Spec.Replicas < 1 {
			return fmt.Errorf("spec.replicas: must be at least 1, got %d", c.Spec.Replicas)
		}

	default:
		return fmt.Errorf("unknown field: %s", fieldPath)
	}

	return nil
}
