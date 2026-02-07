package config

import (
	"os"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestDeploymentConfig(t *testing.T) {
	content, err := os.ReadFile("../../test/fixtures/test_config.yaml")
	if err != nil {
		t.Fatalf("Error reading test config: %v", err)
	}

	var config DeploymentConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		t.Fatalf("Error unmarshaling test config: %v", err)
	}

	// Validate DeploymentConfig fields
	assertEqual(t, config.APIVersion, "kudev.io/v1alpha1", "apiVersion")
	assertEqual(t, config.Kind, "DeploymentConfig", "kind")
	assertEqual(t, config.Metadata.Name, "test-app", "metadata.name")
	assertEqual(t, config.Spec.ImageName, "test-app", "spec.imageName")
	assertEqual(t, config.Spec.Namespace, "default", "spec.namespace")
	assertEqual(t, config.Spec.Replicas, 2, "spec.replicas")
	assertEqual(t, config.Spec.LocalPort, 8080, "spec.localPort")
	assertEqual(t, config.Spec.ServicePort, 8080, "spec.servicePort")
	assertEqual(t, config.Spec.Env[0].Name, "LOG_LEVEL", "spec.env[0].name")
	assertEqual(t, config.Spec.Env[0].Value, "debug", "spec.env[0].value")
}

func assertEqual[T comparable](t *testing.T, got, want T, field string) {
	if got != want {
		t.Errorf("Expected %s to be '%v', got '%v'", field, want, got)
	}
}
