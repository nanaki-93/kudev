package deployer

import (
	"strings"
	"testing"

	"github.com/nanaki-93/kudev/templates"
)

func TestNewRenderer(t *testing.T) {
	renderer, err := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	if renderer == nil {
		t.Fatal("renderer is nil")
	}
}

func TestNewRenderer_InvalidTemplate(t *testing.T) {
	_, err := NewRenderer("{{ .Invalid }", "valid")
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

func TestRenderDeployment(t *testing.T) {
	renderer, err := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := TemplateData{
		AppName:     "test-app",
		Namespace:   "test-ns",
		ImageRef:    "test-app:kudev-12345678",
		ImageHash:   "12345678",
		ServicePort: 8080,
		Replicas:    2,
		Env: []EnvVar{
			{Name: "LOG_LEVEL", Value: "debug"},
		},
	}

	deployment, err := renderer.RenderDeployment(data)
	if err != nil {
		t.Fatalf("RenderDeployment failed: %v", err)
	}

	// Verify deployment fields
	if deployment.Name != "test-app" {
		t.Errorf("Name = %q, want %q", deployment.Name, "test-app")
	}

	if deployment.Namespace != "test-ns" {
		t.Errorf("Namespace = %q, want %q", deployment.Namespace, "test-ns")
	}

	if *deployment.Spec.Replicas != 2 {
		t.Errorf("Replicas = %d, want %d", *deployment.Spec.Replicas, 2)
	}

	// Verify labels
	if deployment.Labels["managed-by"] != "kudev" {
		t.Error("missing managed-by label")
	}

	if deployment.Labels["kudev-hash"] != "12345678" {
		t.Error("missing or incorrect kudev-hash label")
	}

	// Verify container
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(containers))
	}

	if containers[0].Image != "test-app:kudev-12345678" {
		t.Errorf("Image = %q, want %q", containers[0].Image, "test-app:kudev-12345678")
	}

	// Verify env vars
	if len(containers[0].Env) != 1 {
		t.Errorf("expected 1 env var, got %d", len(containers[0].Env))
	}
}

func TestRenderService(t *testing.T) {
	renderer, err := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	data := TemplateData{
		AppName:     "test-app",
		Namespace:   "test-ns",
		ImageRef:    "test-app:latest",
		ImageHash:   "12345678",
		ServicePort: 3000,
		Replicas:    1,
	}

	service, err := renderer.RenderService(data)
	if err != nil {
		t.Fatalf("RenderService failed: %v", err)
	}

	if service.Name != "test-app" {
		t.Errorf("Name = %q, want %q", service.Name, "test-app")
	}

	if len(service.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(service.Spec.Ports))
	}

	if service.Spec.Ports[0].Port != 3000 {
		t.Errorf("Port = %d, want %d", service.Spec.Ports[0].Port, 3000)
	}

	if service.Spec.Selector["app"] != "test-app" {
		t.Error("service selector doesn't match app name")
	}
}

func TestRenderDeployment_InvalidData(t *testing.T) {
	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	data := TemplateData{
		// Missing required fields
	}

	_, err := renderer.RenderDeployment(data)
	if err == nil {
		t.Error("expected error for invalid data")
	}
}

func TestRenderDeploymentYAML(t *testing.T) {
	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	data := TemplateData{
		AppName:     "myapp",
		Namespace:   "default",
		ImageRef:    "myapp:v1",
		ImageHash:   "abc12345",
		ServicePort: 8080,
		Replicas:    1,
	}

	yamlStr, err := renderer.RenderDeploymentYAML(data)
	if err != nil {
		t.Fatalf("RenderDeploymentYAML failed: %v", err)
	}

	// Verify YAML contains expected values
	if !strings.Contains(yamlStr, "name: myapp") {
		t.Error("YAML doesn't contain app name")
	}

	if !strings.Contains(yamlStr, "image: myapp:v1") {
		t.Error("YAML doesn't contain image")
	}
}

func TestRenderAll(t *testing.T) {
	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	data := TemplateData{
		AppName:     "myapp",
		Namespace:   "default",
		ImageRef:    "myapp:v1",
		ImageHash:   "abc12345",
		ServicePort: 8080,
		Replicas:    1,
	}

	combined, err := renderer.RenderAll(data)
	if err != nil {
		t.Fatalf("RenderAll failed: %v", err)
	}

	// Should contain both Deployment and Service
	if !strings.Contains(combined, "kind: Deployment") {
		t.Error("missing Deployment")
	}

	if !strings.Contains(combined, "kind: Service") {
		t.Error("missing Service")
	}

	// Should have document separator
	if !strings.Contains(combined, "---") {
		t.Error("missing YAML document separator")
	}
}
