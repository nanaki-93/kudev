# Task 3.2: Define Template Data Structures

## Overview

This task defines the **data structures** used for template rendering and deployment status tracking. These types form the contract between configuration, rendering, and Kubernetes operations.

**Effort**: ~1-2 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: Phase 1 (Config Types)  
**Files to Create**:
- `pkg/deployer/types.go` — All type definitions

---

## What You're Building

Go types that:
1. **TemplateData** — Data passed to YAML templates
2. **DeploymentStatus** — Current state of deployment
3. **Deployer interface** — Contract for K8s operations
4. **DeploymentOptions** — Input for deployment operations

---

## Complete Implementation

### File Structure

```
pkg/deployer/
├── types.go           ← You'll create this
├── renderer.go        ← Task 3.3
├── deployer.go        ← Task 3.4
├── status.go          ← Task 3.5
└── delete.go          ← Task 3.6
```

### Types Implementation

```go
// pkg/deployer/types.go

package deployer

import (
    "context"
    "time"
    
    "github.com/your-org/kudev/pkg/config"
)

// TemplateData is passed to YAML templates for rendering.
// All fields must match template placeholders exactly.
type TemplateData struct {
    // AppName is the application name used for deployment and service.
    // Maps to {{ .AppName }} in templates.
    AppName string
    
    // Namespace is the Kubernetes namespace for deployment.
    // Maps to {{ .Namespace }} in templates.
    Namespace string
    
    // ImageRef is the full image reference including tag.
    // Example: "myapp:kudev-a1b2c3d4"
    // Maps to {{ .ImageRef }} in templates.
    ImageRef string
    
    // ImageHash is the source code hash (8 characters).
    // Used for tracking deployed version.
    // Maps to {{ .ImageHash }} in templates.
    ImageHash string
    
    // ServicePort is the container port to expose.
    // Maps to {{ .ServicePort }} in templates.
    ServicePort int32
    
    // Replicas is the number of pod replicas.
    // Maps to {{ .Replicas }} in templates.
    Replicas int32
    
    // Env is the list of environment variables.
    // Maps to {{ .Env }} in templates (iterated with range).
    Env []EnvVar
}

// EnvVar represents an environment variable name-value pair.
// Matches the structure expected by templates.
type EnvVar struct {
    // Name is the environment variable name.
    // Must be a valid C identifier (uppercase, underscores).
    Name string
    
    // Value is the environment variable value.
    Value string
}

// DeploymentStatus represents the current state of a deployment.
type DeploymentStatus struct {
    // DeploymentName is the name of the deployment.
    DeploymentName string
    
    // Namespace is the Kubernetes namespace.
    Namespace string
    
    // ReadyReplicas is the number of ready pod replicas.
    ReadyReplicas int32
    
    // DesiredReplicas is the desired number of replicas.
    DesiredReplicas int32
    
    // Status is a human-readable status string.
    // Values: "Running", "Pending", "Degraded", "Failed", "Unknown"
    Status string
    
    // Pods contains status information for each pod.
    Pods []PodStatus
    
    // Message is a helpful status message for the user.
    Message string
    
    // ImageHash is the currently deployed source hash.
    ImageHash string
    
    // LastUpdated is when the deployment was last updated.
    LastUpdated time.Time
}

// PodStatus represents the status of an individual pod.
type PodStatus struct {
    // Name is the pod name.
    Name string
    
    // Status is the pod phase (Running, Pending, Failed, etc).
    Status string
    
    // Ready indicates if the pod is ready to serve traffic.
    Ready bool
    
    // Restarts is the total container restart count.
    Restarts int32
    
    // CreatedAt is when the pod was created.
    CreatedAt time.Time
    
    // Message is additional status info (e.g., crash reason).
    Message string
}

// DeploymentOptions contains input for deployment operations.
type DeploymentOptions struct {
    // Config is the loaded kudev configuration.
    Config *config.DeploymentConfig
    
    // ImageRef is the built image reference (from Phase 2).
    // Example: "myapp:kudev-a1b2c3d4"
    ImageRef string
    
    // ImageHash is the source code hash (from Phase 2).
    ImageHash string
}

// Deployer is the interface for Kubernetes deployment operations.
type Deployer interface {
    // Upsert creates a new deployment or updates an existing one.
    // It also creates/updates the associated Service.
    // Returns the status after deployment.
    Upsert(ctx context.Context, opts DeploymentOptions) (*DeploymentStatus, error)
    
    // Delete removes the deployment and associated service.
    // It only deletes resources with the `managed-by: kudev` label.
    // Safe to call multiple times (idempotent).
    Delete(ctx context.Context, appName, namespace string) error
    
    // Status returns the current deployment status.
    // Returns error if deployment doesn't exist.
    Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error)
}

// StatusCode represents deployment health.
type StatusCode string

const (
    // StatusRunning means all replicas are ready.
    StatusRunning StatusCode = "Running"
    
    // StatusPending means deployment is starting up.
    StatusPending StatusCode = "Pending"
    
    // StatusDegraded means some replicas are not ready.
    StatusDegraded StatusCode = "Degraded"
    
    // StatusFailed means deployment has failed.
    StatusFailed StatusCode = "Failed"
    
    // StatusUnknown means status cannot be determined.
    StatusUnknown StatusCode = "Unknown"
)

// IsHealthy returns true if status indicates healthy deployment.
func (s StatusCode) IsHealthy() bool {
    return s == StatusRunning
}

// String returns the status as a string.
func (s StatusCode) String() string {
    return string(s)
}
```

---

## Type Relationships

```
DeploymentOptions
    │
    ├── Config (*config.DeploymentConfig)
    │      └── From .kudev.yaml via config.LoadConfig()
    │
    ├── ImageRef (string)
    │      └── From builder.Build() → ImageRef.FullRef
    │
    └── ImageHash (string)
           └── From hash.Calculator.Calculate()


TemplateData (rendered from DeploymentOptions + Config)
    │
    ├── AppName     ← Config.Metadata.Name
    ├── Namespace   ← Config.Spec.Namespace
    ├── ImageRef    ← DeploymentOptions.ImageRef
    ├── ImageHash   ← DeploymentOptions.ImageHash
    ├── ServicePort ← Config.Spec.ServicePort
    ├── Replicas    ← Config.Spec.Replicas
    └── Env[]       ← Config.Spec.Env


DeploymentStatus (returned from K8s API queries)
    │
    ├── ReadyReplicas, DesiredReplicas ← from Deployment status
    ├── Status (StatusCode)            ← computed from replicas
    └── Pods[]                         ← from Pod list query
```

---

## Helper Functions

Add helper functions for common operations:

```go
// NewTemplateData creates TemplateData from DeploymentOptions.
// This is the bridge between config and templates.
func NewTemplateData(opts DeploymentOptions) TemplateData {
    // Convert config.EnvVar to deployer.EnvVar
    var envVars []EnvVar
    for _, e := range opts.Config.Spec.Env {
        envVars = append(envVars, EnvVar{
            Name:  e.Name,
            Value: e.Value,
        })
    }
    
    return TemplateData{
        AppName:     opts.Config.Metadata.Name,
        Namespace:   opts.Config.Spec.Namespace,
        ImageRef:    opts.ImageRef,
        ImageHash:   opts.ImageHash,
        ServicePort: opts.Config.Spec.ServicePort,
        Replicas:    opts.Config.Spec.Replicas,
        Env:         envVars,
    }
}

// Validate checks that TemplateData has all required fields.
func (td TemplateData) Validate() error {
    var errors []string
    
    if td.AppName == "" {
        errors = append(errors, "AppName is required")
    }
    if td.Namespace == "" {
        errors = append(errors, "Namespace is required")
    }
    if td.ImageRef == "" {
        errors = append(errors, "ImageRef is required")
    }
    if td.ServicePort <= 0 {
        errors = append(errors, "ServicePort must be positive")
    }
    if td.Replicas <= 0 {
        errors = append(errors, "Replicas must be positive")
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("invalid TemplateData: %s", strings.Join(errors, ", "))
    }
    
    return nil
}

// IsReady returns true if deployment has all replicas ready.
func (ds *DeploymentStatus) IsReady() bool {
    return ds.ReadyReplicas >= ds.DesiredReplicas && ds.DesiredReplicas > 0
}

// Summary returns a one-line status summary.
func (ds *DeploymentStatus) Summary() string {
    return fmt.Sprintf("%s: %d/%d replicas ready (%s)",
        ds.DeploymentName,
        ds.ReadyReplicas,
        ds.DesiredReplicas,
        ds.Status,
    )
}
```

---

## Testing Types

### Unit Tests

```go
// pkg/deployer/types_test.go

package deployer

import (
    "testing"
    
    "github.com/your-org/kudev/pkg/config"
)

func TestNewTemplateData(t *testing.T) {
    cfg := &config.DeploymentConfig{
        Metadata: config.ConfigMetadata{
            Name: "myapp",
        },
        Spec: config.DeploymentSpec{
            Namespace:   "production",
            ServicePort: 8080,
            Replicas:    3,
            Env: []config.EnvVar{
                {Name: "LOG_LEVEL", Value: "info"},
            },
        },
    }
    
    opts := DeploymentOptions{
        Config:    cfg,
        ImageRef:  "myapp:kudev-abc12345",
        ImageHash: "abc12345",
    }
    
    data := NewTemplateData(opts)
    
    if data.AppName != "myapp" {
        t.Errorf("AppName = %q, want %q", data.AppName, "myapp")
    }
    
    if data.Namespace != "production" {
        t.Errorf("Namespace = %q, want %q", data.Namespace, "production")
    }
    
    if data.ImageRef != "myapp:kudev-abc12345" {
        t.Errorf("ImageRef = %q, want %q", data.ImageRef, "myapp:kudev-abc12345")
    }
    
    if len(data.Env) != 1 {
        t.Errorf("len(Env) = %d, want 1", len(data.Env))
    }
}

func TestTemplateDataValidate(t *testing.T) {
    tests := []struct {
        name    string
        data    TemplateData
        wantErr bool
    }{
        {
            name: "valid data",
            data: TemplateData{
                AppName:     "myapp",
                Namespace:   "default",
                ImageRef:    "myapp:latest",
                ImageHash:   "abc12345",
                ServicePort: 8080,
                Replicas:    1,
            },
            wantErr: false,
        },
        {
            name: "missing AppName",
            data: TemplateData{
                Namespace:   "default",
                ImageRef:    "myapp:latest",
                ServicePort: 8080,
                Replicas:    1,
            },
            wantErr: true,
        },
        {
            name: "invalid port",
            data: TemplateData{
                AppName:     "myapp",
                Namespace:   "default",
                ImageRef:    "myapp:latest",
                ServicePort: 0,
                Replicas:    1,
            },
            wantErr: true,
        },
        {
            name: "invalid replicas",
            data: TemplateData{
                AppName:     "myapp",
                Namespace:   "default",
                ImageRef:    "myapp:latest",
                ServicePort: 8080,
                Replicas:    0,
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.data.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}

func TestDeploymentStatusIsReady(t *testing.T) {
    tests := []struct {
        name    string
        status  DeploymentStatus
        want    bool
    }{
        {
            name:   "all ready",
            status: DeploymentStatus{ReadyReplicas: 3, DesiredReplicas: 3},
            want:   true,
        },
        {
            name:   "some ready",
            status: DeploymentStatus{ReadyReplicas: 2, DesiredReplicas: 3},
            want:   false,
        },
        {
            name:   "none ready",
            status: DeploymentStatus{ReadyReplicas: 0, DesiredReplicas: 3},
            want:   false,
        },
        {
            name:   "zero desired",
            status: DeploymentStatus{ReadyReplicas: 0, DesiredReplicas: 0},
            want:   false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := tt.status.IsReady(); got != tt.want {
                t.Errorf("IsReady() = %v, want %v", got, tt.want)
            }
        })
    }
}

func TestStatusCodeIsHealthy(t *testing.T) {
    tests := []struct {
        status  StatusCode
        healthy bool
    }{
        {StatusRunning, true},
        {StatusPending, false},
        {StatusDegraded, false},
        {StatusFailed, false},
        {StatusUnknown, false},
    }
    
    for _, tt := range tests {
        t.Run(string(tt.status), func(t *testing.T) {
            if got := tt.status.IsHealthy(); got != tt.healthy {
                t.Errorf("IsHealthy() = %v, want %v", got, tt.healthy)
            }
        })
    }
}
```

---

## How Types Connect to Other Tasks

```
Task 3.2 (Types) ← You are here
    │
    ├─► Task 3.3 (Renderer)
    │   - Uses TemplateData for rendering
    │
    ├─► Task 3.4 (Deployer - Upsert)
    │   - Uses DeploymentOptions as input
    │   - Implements Deployer interface
    │
    ├─► Task 3.5 (Status)
    │   - Returns DeploymentStatus
    │   - Uses StatusCode enum
    │
    └─► Task 3.6 (Delete)
        - Uses Deployer interface
```

---

## Checklist for Task 3.2

- [ ] Create `pkg/deployer/types.go`
- [ ] Define `TemplateData` struct with all fields
- [ ] Define `EnvVar` struct
- [ ] Define `DeploymentStatus` struct
- [ ] Define `PodStatus` struct
- [ ] Define `DeploymentOptions` struct
- [ ] Define `Deployer` interface
- [ ] Define `StatusCode` type with constants
- [ ] Implement `NewTemplateData()` helper
- [ ] Implement `TemplateData.Validate()` method
- [ ] Implement `DeploymentStatus.IsReady()` method
- [ ] Implement `DeploymentStatus.Summary()` method
- [ ] Implement `StatusCode.IsHealthy()` method
- [ ] Add doc comments for all types and fields
- [ ] Create `pkg/deployer/types_test.go`
- [ ] Run `go test ./pkg/deployer -v`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Mismatched template field names
```go
// Wrong - template uses {{ .AppName }} but struct has Name
type TemplateData struct {
    Name string  // ← Wrong!
}

// Right
type TemplateData struct {
    AppName string  // ← Matches template
}
```

❌ **Mistake 2**: Not handling empty Env slice
```go
// Templates must handle empty Env gracefully
// Use: {{- if .Env }} ... {{- end }}
```

❌ **Mistake 3**: Using int instead of int32
```go
// Wrong - K8s uses int32
Replicas int

// Right
Replicas int32
```

---

## Next Steps

1. **Complete this task** ← You are here
2. Move to **Task 3.3** → Implement Template Rendering
3. Renderer will use TemplateData to generate K8s objects

---

## References

- [Go Struct Tags](https://golang.org/pkg/reflect/#StructTag)
- [K8s API Types](https://pkg.go.dev/k8s.io/api)
- [Interface Best Practices](https://go.dev/doc/effective_go#interfaces)

