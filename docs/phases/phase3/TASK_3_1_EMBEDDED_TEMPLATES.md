# Task 3.1: Create Embedded YAML Templates

## Overview

This task creates **embedded YAML templates** for Kubernetes Deployment and Service resources. Templates are embedded directly in the binary using Go's `//go:embed` directive, eliminating external file dependencies.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: None (pure Go, YAML knowledge)  
**Files to Create**:
- `templates/deployment.yaml` — Deployment template
- `templates/service.yaml` — Service template
- `templates/embed.go` — Go embed declarations

---

## What You're Building

YAML templates that:
1. **Define** standard K8s Deployment and Service structure
2. **Use** Go template placeholders for dynamic values
3. **Include** proper labels for resource management
4. **Embed** in binary for zero-dependency distribution

---

## The Problem This Solves

### Why Embedded Templates?

| Approach | Pros | Cons |
|----------|------|------|
| User-provided YAML | Maximum flexibility | Complex, error-prone |
| Helm charts | Feature-rich | Heavy dependency, overkill |
| Kustomize | Industry standard | Additional tooling |
| **Embedded templates** | Simple, portable | Less flexible |

**For kudev, embedded templates are ideal**:
- Single binary distribution
- Sensible defaults work for 90% of cases
- Future: Allow overrides if needed
- `--dry-run` can show generated manifests

---

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                templates/                            │
├─────────────────────────────────────────────────────┤
│  deployment.yaml    (YAML with {{.Placeholders}})   │
│  service.yaml       (YAML with {{.Placeholders}})   │
│  embed.go           (//go:embed declarations)       │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│              pkg/deployer/renderer.go                │
│              (Parses and renders templates)          │
└─────────────────────────────────────────────────────┘
```

---

## Complete Implementation

### File Structure

```
templates/
├── deployment.yaml   ← Deployment template
├── service.yaml      ← Service template
└── embed.go          ← Go embed declarations
```

### Deployment Template

```yaml
# templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .AppName }}
  namespace: {{ .Namespace }}
  labels:
    app: {{ .AppName }}
    managed-by: kudev
    kudev-hash: {{ .ImageHash }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .AppName }}
  template:
    metadata:
      labels:
        app: {{ .AppName }}
        managed-by: kudev
    spec:
      containers:
      - name: {{ .AppName }}
        image: {{ .ImageRef }}
        ports:
        - containerPort: {{ .ServicePort }}
          name: http
        {{- if .Env }}
        env:
        {{- range .Env }}
        - name: {{ .Name }}
          value: "{{ .Value }}"
        {{- end }}
        {{- end }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
          requests:
            cpu: "100m"
            memory: "128Mi"
```

### Service Template

```yaml
# templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .AppName }}
  namespace: {{ .Namespace }}
  labels:
    app: {{ .AppName }}
    managed-by: kudev
spec:
  type: ClusterIP
  ports:
  - port: {{ .ServicePort }}
    targetPort: {{ .ServicePort }}
    protocol: TCP
    name: http
  selector:
    app: {{ .AppName }}
```

### Embed Declarations

```go
// templates/embed.go

package templates

import (
    _ "embed"
)

// DeploymentTemplate is the embedded Deployment YAML template.
//
//go:embed deployment.yaml
var DeploymentTemplate string

// ServiceTemplate is the embedded Service YAML template.
//
//go:embed service.yaml
var ServiceTemplate string
```

---

## Template Placeholders Explained

### Deployment Placeholders

| Placeholder | Type | Description | Example |
|-------------|------|-------------|---------|
| `{{ .AppName }}` | string | Application/deployment name | `myapp` |
| `{{ .Namespace }}` | string | K8s namespace | `default` |
| `{{ .ImageRef }}` | string | Full image reference | `myapp:kudev-a1b2c3d4` |
| `{{ .ImageHash }}` | string | Source code hash | `a1b2c3d4` |
| `{{ .Replicas }}` | int32 | Number of replicas | `2` |
| `{{ .ServicePort }}` | int32 | Container port | `8080` |
| `{{ .Env }}` | []EnvVar | Environment variables | `[{Name: "LOG_LEVEL", Value: "info"}]` |

### Service Placeholders

| Placeholder | Type | Description | Example |
|-------------|------|-------------|---------|
| `{{ .AppName }}` | string | Service name | `myapp` |
| `{{ .Namespace }}` | string | K8s namespace | `default` |
| `{{ .ServicePort }}` | int32 | Service port | `8080` |

---

## Critical Labels Explained

### `managed-by: kudev`

Identifies resources created by kudev:
```yaml
labels:
  managed-by: kudev
```

**Purpose**:
- Safe deletion (only delete kudev resources)
- Identify kudev resources with `kubectl get all -l managed-by=kudev`
- Prevent accidental modification of non-kudev resources

### `kudev-hash: {hash}`

Tracks deployed source code version:
```yaml
labels:
  kudev-hash: a1b2c3d4
```

**Purpose**:
- Detect if redeployment needed
- Track which version is running
- Debug deployments

### `app: {appname}`

Standard K8s label for pod selection:
```yaml
labels:
  app: myapp
selector:
  matchLabels:
    app: myapp
```

**Purpose**:
- Service targets pods with this label
- Log tailing finds pods by label
- Standard K8s convention

---

## Template Syntax Deep Dive

### Conditional Rendering

Only render env section if environment variables exist:
```yaml
{{- if .Env }}
env:
{{- range .Env }}
- name: {{ .Name }}
  value: "{{ .Value }}"
{{- end }}
{{- end }}
```

**Note**: The `-` in `{{-` trims whitespace before the tag.

### Range Loops

Iterate over environment variables:
```yaml
{{- range .Env }}
- name: {{ .Name }}
  value: "{{ .Value }}"
{{- end }}
```

### Quoting Values

Always quote string values to handle special characters:
```yaml
value: "{{ .Value }}"
```

---

## Testing Templates

### Manual Template Test

```go
package main

import (
    "bytes"
    "fmt"
    "text/template"
    
    "github.com/your-org/kudev/templates"
)

type TemplateData struct {
    AppName     string
    Namespace   string
    ImageRef    string
    ImageHash   string
    Replicas    int32
    ServicePort int32
    Env         []EnvVar
}

type EnvVar struct {
    Name  string
    Value string
}

func main() {
    data := TemplateData{
        AppName:     "myapp",
        Namespace:   "default",
        ImageRef:    "myapp:kudev-a1b2c3d4",
        ImageHash:   "a1b2c3d4",
        Replicas:    2,
        ServicePort: 8080,
        Env: []EnvVar{
            {Name: "LOG_LEVEL", Value: "debug"},
            {Name: "PORT", Value: "8080"},
        },
    }
    
    tpl, err := template.New("deployment").Parse(templates.DeploymentTemplate)
    if err != nil {
        panic(err)
    }
    
    var buf bytes.Buffer
    if err := tpl.Execute(&buf, data); err != nil {
        panic(err)
    }
    
    fmt.Println(buf.String())
}
```

### Unit Tests

```go
// templates/embed_test.go

package templates

import (
    "bytes"
    "testing"
    "text/template"
    
    "sigs.k8s.io/yaml"
    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
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
```

---

## Checklist for Task 3.1

- [ ] Create `templates/` directory
- [ ] Create `templates/deployment.yaml` with all placeholders
- [ ] Create `templates/service.yaml` with all placeholders
- [ ] Create `templates/embed.go` with `//go:embed` directives
- [ ] Add `managed-by: kudev` label to all resources
- [ ] Add `kudev-hash` label to Deployment
- [ ] Add `app` label for pod selection
- [ ] Set `imagePullPolicy: IfNotPresent`
- [ ] Add default resource limits
- [ ] Write tests in `templates/embed_test.go`
- [ ] Verify templates render valid K8s YAML
- [ ] Run `go build ./templates`
- [ ] Run `go test ./templates -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Forgetting the hyphen in template tags
```yaml
# Wrong - leaves extra whitespace
{{ if .Env }}
env:
{{ end }}

# Right - trims whitespace
{{- if .Env }}
env:
{{- end }}
```

❌ **Mistake 2**: Not quoting string values
```yaml
# Wrong - breaks on special characters
value: {{ .Value }}

# Right - handles special chars
value: "{{ .Value }}"
```

❌ **Mistake 3**: Missing selector match
```yaml
# Wrong - selector doesn't match pod labels
spec:
  selector:
    matchLabels:
      name: {{ .AppName }}  # "name" not "app"
  template:
    metadata:
      labels:
        app: {{ .AppName }}  # Uses "app"!

# Right - matching labels
spec:
  selector:
    matchLabels:
      app: {{ .AppName }}
  template:
    metadata:
      labels:
        app: {{ .AppName }}
```

❌ **Mistake 4**: Using `Always` image pull policy
```yaml
# Wrong - always pulls from registry (fails for local images)
imagePullPolicy: Always

# Right - uses local images
imagePullPolicy: IfNotPresent
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 3.2** → Define Template Data Structures
3. Renderer will use these templates with TemplateData

---

## References

- [Go Embed](https://pkg.go.dev/embed)
- [Go Templates](https://pkg.go.dev/text/template)
- [Kubernetes Deployment Spec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#deploymentspec-v1-apps)
- [Kubernetes Service Spec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#servicespec-v1-core)

