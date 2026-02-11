// templates/embed_test.go

package templates

import (
	"bytes"
	"testing"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type testTemplateData struct {
	AppName     string
	Namespace   string
	ImageRef    string
	ImageHash   string
	Replicas    int32
	ServicePort int32
	Env         []testEnvVar
}

type testEnvVar struct {
	Name  string
	Value string
}

func TestDeploymentTemplateValid(t *testing.T) {
	data := testTemplateData{
		AppName:     "test-app",
		Namespace:   "test-ns",
		ImageRef:    "test-app:kudev-12345678",
		ImageHash:   "12345678",
		Replicas:    1,
		ServicePort: 8080,
	}

	tpl, err := template.New("deployment").Parse(DeploymentTemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	// Verify it's valid K8s YAML
	var deployment appsv1.Deployment
	if err := yaml.Unmarshal(buf.Bytes(), &deployment); err != nil {
		t.Fatalf("invalid deployment YAML: %v", err)
	}

	// Verify values
	if deployment.Name != "test-app" {
		t.Errorf("name = %q, want %q", deployment.Name, "test-app")
	}

	if deployment.Namespace != "test-ns" {
		t.Errorf("namespace = %q, want %q", deployment.Namespace, "test-ns")
	}

	if deployment.Labels["managed-by"] != "kudev" {
		t.Error("missing managed-by label")
	}
}

func TestServiceTemplateValid(t *testing.T) {
	data := testTemplateData{
		AppName:     "test-app",
		Namespace:   "test-ns",
		ServicePort: 8080,
	}

	tpl, err := template.New("service").Parse(ServiceTemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	var service corev1.Service
	if err := yaml.Unmarshal(buf.Bytes(), &service); err != nil {
		t.Fatalf("invalid service YAML: %v", err)
	}

	if service.Name != "test-app" {
		t.Errorf("name = %q, want %q", service.Name, "test-app")
	}

	if service.Spec.Selector["app"] != "test-app" {
		t.Error("service selector doesn't match app name")
	}
}

func TestDeploymentTemplateWithEnv(t *testing.T) {
	data := testTemplateData{
		AppName:     "test-app",
		Namespace:   "default",
		ImageRef:    "test-app:latest",
		ImageHash:   "12345678",
		Replicas:    1,
		ServicePort: 8080,
		Env: []testEnvVar{
			{Name: "LOG_LEVEL", Value: "debug"},
			{Name: "DATABASE_URL", Value: "postgres://localhost/db"},
		},
	}

	tpl, err := template.New("deployment").Parse(DeploymentTemplate)
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	var deployment appsv1.Deployment
	if err := yaml.Unmarshal(buf.Bytes(), &deployment); err != nil {
		t.Fatalf("invalid deployment YAML: %v", err)
	}

	envVars := deployment.Spec.Template.Spec.Containers[0].Env
	if len(envVars) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(envVars))
	}
}

func TestTemplatesAreEmbedded(t *testing.T) {
	if DeploymentTemplate == "" {
		t.Error("DeploymentTemplate is empty")
	}

	if ServiceTemplate == "" {
		t.Error("ServiceTemplate is empty")
	}
}
