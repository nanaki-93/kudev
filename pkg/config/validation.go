package config

import (
	"fmt"
	"regexp"
)

func (c *DeploymentConfig) Validate() error {
	if c.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}
	if c.Spec.ImageName == "" {
		return fmt.Errorf("spec.imageName is required")
	}
	if c.Spec.DockerFilePath == "" {
		return fmt.Errorf("spec.dockerFilePath is required")
	}

	if !isValidDNS(c.Metadata.Name) {
		return fmt.Errorf("metadata.name must be valid DNS name")
	}
	if c.Spec.LocalPort < 1 || c.Spec.LocalPort > 65535 {
		return fmt.Errorf("spec.localPort must be between 1 and 65535")
	}

	if c.Spec.ServicePort < 1 || c.Spec.ServicePort > 65535 {
		return fmt.Errorf("spec.localPort must be between 1 and 65535")
	}

	if c.Spec.Replicas < 1 {
		return fmt.Errorf("spec.replicas must be at least 1")
	}
	if c.Spec.Namespace == "" {
		return fmt.Errorf("spec.namespace is required")
	}
	if !isValidDNS(c.Spec.Namespace) {
		return fmt.Errorf("spec.namespace must be valid DNS name")
	}
	return nil

}

func isValidDNS(dns string) bool {

	match, _ := regexp.MatchString("^[a-z0-9]([-a-z0-9]*[a-z0-9])?$", dns)
	return match
}
