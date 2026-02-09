# Task 3.3: Implement Template Rendering

## Overview

This task implements the **template rendering engine** that takes TemplateData and produces valid Kubernetes Deployment and Service objects.

**Effort**: ~2-3 hours  
**Complexity**: 🟡 Intermediate  
**Dependencies**: Task 3.1 (Templates), Task 3.2 (Types)  
**Files to Create**:
- `pkg/deployer/renderer.go` — Template rendering logic
- `pkg/deployer/renderer_test.go` — Tests

---

## What You're Building

A renderer that:
1. **Parses** Go templates with custom functions
2. **Executes** templates with TemplateData
3. **Unmarshals** resulting YAML to K8s objects
4. **Validates** output is well-formed
5. **Returns** typed K8s objects ready for API calls

---

## Complete Implementation

```go
// pkg/deployer/renderer.go

package deployer

import (
    "bytes"
    "fmt"
    "text/template"
    
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/yaml"
)

// Renderer handles YAML template rendering.
type Renderer struct {
    deploymentTpl *template.Template
    serviceTpl    *template.Template
}

// templateFuncs provides custom functions for templates.
var templateFuncs = template.FuncMap{
    // quote wraps a string in double quotes
    "quote": func(s string) string {
        return fmt.Sprintf("%q", s)
    },
    // default returns the default value if the input is empty
    "default": func(defaultVal, val interface{}) interface{} {
        if val == nil || val == "" {
            return defaultVal
        }
        return val
    },
}

// NewRenderer creates a new template renderer.
// deploymentTpl and serviceTpl are the raw template strings (from go:embed).
func NewRenderer(deploymentTpl, serviceTpl string) (*Renderer, error) {
    depTpl, err := template.New("deployment").
        Funcs(templateFuncs).
        Parse(deploymentTpl)
    if err != nil {
        return nil, fmt.Errorf("failed to parse deployment template: %w", err)
    }
    
    svcTpl, err := template.New("service").
        Funcs(templateFuncs).
        Parse(serviceTpl)
    if err != nil {
        return nil, fmt.Errorf("failed to parse service template: %w", err)
    }
    
    return &Renderer{
        deploymentTpl: depTpl,
        serviceTpl:    svcTpl,
    }, nil
}

// RenderDeployment renders the Deployment template with the given data.
// Returns a typed Kubernetes Deployment object.
func (r *Renderer) RenderDeployment(data TemplateData) (*appsv1.Deployment, error) {
    // Validate input
    if err := data.Validate(); err != nil {
        return nil, fmt.Errorf("invalid template data: %w", err)
    }
    
    // Execute template
    var buf bytes.Buffer
    if err := r.deploymentTpl.Execute(&buf, data); err != nil {
        return nil, fmt.Errorf("failed to execute deployment template: %w", err)
    }
    
    // Parse YAML into K8s object
    deployment := &appsv1.Deployment{}
    if err := yaml.Unmarshal(buf.Bytes(), deployment); err != nil {
        return nil, fmt.Errorf("failed to unmarshal deployment YAML: %w\nRendered YAML:\n%s", 
            err, buf.String())
    }
    
    return deployment, nil
}

// RenderService renders the Service template with the given data.
// Returns a typed Kubernetes Service object.
func (r *Renderer) RenderService(data TemplateData) (*corev1.Service, error) {
    // Validate input
    if err := data.Validate(); err != nil {
        return nil, fmt.Errorf("invalid template data: %w", err)
    }
    
    // Execute template
    var buf bytes.Buffer
    if err := r.serviceTpl.Execute(&buf, data); err != nil {
        return nil, fmt.Errorf("failed to execute service template: %w", err)
    }
    
    // Parse YAML into K8s object
    service := &corev1.Service{}
    if err := yaml.Unmarshal(buf.Bytes(), service); err != nil {
        return nil, fmt.Errorf("failed to unmarshal service YAML: %w\nRendered YAML:\n%s",
            err, buf.String())
    }
    
    return service, nil
}

// RenderDeploymentYAML renders the Deployment template and returns raw YAML.
// Useful for --dry-run output.
func (r *Renderer) RenderDeploymentYAML(data TemplateData) (string, error) {
    if err := data.Validate(); err != nil {
        return "", fmt.Errorf("invalid template data: %w", err)
    }
    
    var buf bytes.Buffer
    if err := r.deploymentTpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute deployment template: %w", err)
    }
    
    return buf.String(), nil
}

// RenderServiceYAML renders the Service template and returns raw YAML.
// Useful for --dry-run output.
func (r *Renderer) RenderServiceYAML(data TemplateData) (string, error) {
    if err := data.Validate(); err != nil {
        return "", fmt.Errorf("invalid template data: %w", err)
    }
    
    var buf bytes.Buffer
    if err := r.serviceTpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute service template: %w", err)
    }
    
    return buf.String(), nil
}

// RenderAll renders both Deployment and Service, returning raw YAML.
// Useful for --dry-run to show complete manifests.
func (r *Renderer) RenderAll(data TemplateData) (string, error) {
    depYAML, err := r.RenderDeploymentYAML(data)
    if err != nil {
        return "", err
    }
    
    svcYAML, err := r.RenderServiceYAML(data)
    if err != nil {
        return "", err
    }
    
    // Combine with YAML document separator
    return fmt.Sprintf("%s---\n%s", depYAML, svcYAML), nil
}
```

---

## Key Implementation Details

### 1. Template Functions

Add custom functions for templates:
```go
var templateFuncs = template.FuncMap{
    "quote": func(s string) string {
        return fmt.Sprintf("%q", s)
    },
}
```

Usage in template:
```yaml
value: {{ .Value | quote }}
# Renders: value: "my value with spaces"
```

### 2. Error Context

Include rendered YAML in error messages for debugging:
```go
if err := yaml.Unmarshal(buf.Bytes(), deployment); err != nil {
    return nil, fmt.Errorf("failed to unmarshal: %w\nRendered YAML:\n%s", 
        err, buf.String())
}
```

### 3. sigs.k8s.io/yaml

Use the K8s YAML library (not gopkg.in/yaml.v3):
```go
import "sigs.k8s.io/yaml"

// This handles K8s-specific YAML quirks
yaml.Unmarshal(data, &deployment)
```

### 4. Input Validation

Always validate before rendering:
```go
if err := data.Validate(); err != nil {
    return nil, fmt.Errorf("invalid template data: %w", err)
}
```

---

## Testing the Renderer

```go
// pkg/deployer/renderer_test.go

package deployer

import (
    "strings"
    "testing"
    
    "github.com/your-org/kudev/templates"
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
```

---

## Checklist for Task 3.3

- [ ] Create `pkg/deployer/renderer.go`
- [ ] Implement `Renderer` struct
- [ ] Implement `NewRenderer()` constructor
- [ ] Add `templateFuncs` with `quote` function
- [ ] Implement `RenderDeployment()` method
- [ ] Implement `RenderService()` method
- [ ] Implement `RenderDeploymentYAML()` method
- [ ] Implement `RenderServiceYAML()` method
- [ ] Implement `RenderAll()` method
- [ ] Validate input data before rendering
- [ ] Include rendered YAML in error messages
- [ ] Create `pkg/deployer/renderer_test.go`
- [ ] Test valid rendering
- [ ] Test invalid template handling
- [ ] Test invalid data handling
- [ ] Test YAML output
- [ ] Run `go test ./pkg/deployer -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Using wrong YAML library
```go
// Wrong - doesn't handle K8s specifics
import "gopkg.in/yaml.v3"

// Right - K8s standard
import "sigs.k8s.io/yaml"
```

❌ **Mistake 2**: Not adding template functions before parsing
```go
// Wrong - parse before adding funcs
tpl, _ := template.New("x").Parse(tplStr)
tpl.Funcs(funcs)  // Too late!

// Right - add funcs first
tpl, _ := template.New("x").Funcs(funcs).Parse(tplStr)
```

❌ **Mistake 3**: Swallowing parse errors
```go
// Wrong - no error context
return nil, err

// Right - include context
return nil, fmt.Errorf("failed to parse: %w\nYAML:\n%s", err, buf.String())
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 3.4** → Implement Deployer with Upsert Logic
3. Deployer will use Renderer to generate K8s objects

---

## References

- [Go Templates](https://pkg.go.dev/text/template)
- [sigs.k8s.io/yaml](https://pkg.go.dev/sigs.k8s.io/yaml)
- [K8s API Types](https://pkg.go.dev/k8s.io/api)

