# Task 1.1: Define Configuration Types

## Overview

This task establishes the Go type definitions that represent the `.kudev.yaml` configuration file. These types are the **foundation** for all subsequent operations: loading, validation, manipulation, and serialization.

**Effort**: ~2-3 hours  
**Complexity**: 🟢 Beginner-Friendly  
**Dependencies**: None (pure Go types)

---

## What You're Building

A set of Go structs that:
1. **Represent** the YAML structure in idiomatic Go
2. **Marshal/Unmarshal** to/from YAML without data loss
3. **Follow K8s conventions** for API versioning and metadata
4. **Support** both YAML and JSON (for future API endpoints)
5. **Document** each field with clear intent

---

## Complete Schema Breakdown

### Why This Schema?

```yaml
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
```
- **`apiVersion`**: Allows versioning the config format
- **`kind`**: Identifies the resource type (enables future flexibility like `ServiceConfig`)
- This matches **K8s resource format** for consistency

```yaml
metadata:
  name: myapp
```
- **`name`**: Used as:
  - Deployment name in K8s
  - Docker image base name
  - Label identifier
- Must be DNS-1123 compliant (validated in Task 1.2)

```yaml
spec:
  imageName: myapp
  dockerfilePath: ./Dockerfile
```
- **`imageName`**: Container image name (without registry prefix)
  - Example: `myapp` not `docker.io/user/myapp`
  - Registry added during build phase
- **`dockerfilePath`**: Relative to project root
  - Used by build system to locate Dockerfile
  - Examples: `./Dockerfile`, `./docker/Dockerfile.dev`

```yaml
  namespace: default
  replicas: 1
```
- **`namespace`**: K8s namespace where deployment goes
  - Default: `default`, but explicit is better
- **`replicas`**: Number of pod replicas (int32 like K8s)
  - Minimum: 1 (enforced in validation)

```yaml
  localPort: 8080
  servicePort: 8080
```
- **`localPort`**: Host machine port for port-forwarding
  - Used by: `kubectl port-forward :localPort`
- **`servicePort`**: Container port
  - Exposed by container in Dockerfile EXPOSE
  - Used by K8s Service

```yaml
  env:
    - name: LOG_LEVEL
      value: info
```
- **`env`**: Environment variables injected into containers
- Follows K8s convention (same structure as Pod spec)
- Used by Deployment manifest generation (Phase 3)

```yaml
  kubeContext: docker-desktop
  buildContextExclusions:
    - .git
    - node_modules
```
- **`kubeContext`** (optional): Pin to specific K8s context
  - If set: validate current context matches
  - If unset: use whitelist validation
  - Prevents: "deployed to wrong cluster" accidents

- **`buildContextExclusions`** (optional): Paths to exclude from Docker build
  - Improves build speed and image size
  - Default: `.git`, `node_modules`, `.kudev.yaml` (auto-added)

---

## Go Type Definitions

### File Structure

```
pkg/config/
├── types.go           ← You'll create this
├── validation.go      ← Task 1.2
├── loader.go          ← Task 1.3
├── errors.go          ← Task 1.2
└── types_test.go      ← Tests
```

## Critical Design Decisions Explained

### Decision 1: Using `int32` for Replicas, Ports

**Question**: Why int32 instead of int?

**Answer**: K8s API uses `int32`:
- Deployment.spec.replicas is int32
- Service.spec.ports[].port is int32
- Using same type prevents conversion bugs
- Matches industry standard

### Decision 2: Separate `imageName` from `metadata.name`

**Question**: Why not just use metadata.name?

**Answer**: They serve different purposes:
- `metadata.name`: Deployment/Service name in K8s
  - Must be DNS-1123 (lowercase, hyphens)
  - Must be globally unique in namespace
  - Examples: "my-app", "api-gateway"

- `imageName`: Docker image base name
  - Follows Docker naming (similar but not identical)
  - Might differ from deployment name
  - Example: metadata.name="my-app", imageName="myapp"

**Example scenario**:
```yaml
metadata:
  name: my-production-app
spec:
  imageName: prod-app  # Different! More concise
```

### Decision 3: `env` as List not Map

**Question**: Why `env: [{name, value}]` not `env: {LOG_LEVEL: info}`?

**Answer**: 
1. **K8s standard**: Pod spec uses list format
2. **Validation**: Can validate each EnvVar individually
3. **Future**: Allows valueFrom (ConfigMaps, Secrets)
4. **Ordering**: Preserves order if needed

---

## How Types Connect to Other Tasks

```
Task 1.1 (Types)
    ↓
Task 1.2 (Validation) - validates types
    ↓
Task 1.3 (Loader) - loads YAML → types
    ↓
Task 1.4 (Kubeconfig) - accesses Spec.KubeContext
    ↓
Phase 2 - BuildConfig uses Spec.ImageName, DockerfilePath
    ↓
Phase 3 - Manifest generation uses Spec.Replicas, Env, ServicePort
```

---

## Testing Your Types (Preview of Task 1.2)

### Quick Manual Test

Create `test_config.yaml`:
```yaml
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: test-app
spec:
  imageName: test-app
  dockerfilePath: ./Dockerfile
  namespace: default
  replicas: 2
  localPort: 8080
  servicePort: 8080
  env:
    - name: LOG_LEVEL
      value: debug
```

Test it:
```bash
# In your main/test code
content, _ := os.ReadFile("test_config.yaml")
var cfg config.DeploymentConfig
yaml.Unmarshal(content, &cfg)
fmt.Printf("%+v\n", cfg)
```

Expected output shows all fields populated correctly.

### Type Safety Test

```go
// Should compile (same type)
var x int32 = cfg.Spec.Replicas

// Should NOT compile (type mismatch)
var y int = cfg.Spec.Replicas  // ← Compile error
```

---

## Dependencies & Imports

Add to your `go.mod`:
```
require (
    sigs.k8s.io/yaml v1.3.0
)
```

In `types.go`:
```go
package config

// No imports needed!
// YAML tags are just strings
// Marshaling happens in loader.go
```

---

## Checklist for Task 1.1

- [X] Create `pkg/config/types.go`
- [X] Define `DeploymentConfig` struct with all fields
- [X] Define `ConfigMetadata` struct
- [X] Define `DeploymentSpec` struct
- [X] Define `EnvVar` struct
- [X] Add YAML tags: `yaml:"fieldName"`
- [X] Add JSON tags: `json:"fieldName"`
- [X] Add `omitempty` to optional fields
- [X] Write doc comments for all types
- [X] Write doc comments for all fields
- [X] Create `NewDeploymentConfig()` helper
- [X] Run `go fmt ./pkg/config`
- [X] Verify no compilation errors: `go build ./pkg/config`

---

## Common Mistakes to Avoid

❌ **Mistake 1**: Forgetting YAML tags
```go
// Wrong - will not unmarshal from YAML
type DeploymentSpec struct {
    ImageName string  // ← no tag!
}

// Right
type DeploymentSpec struct {
    ImageName string `yaml:"imageName" json:"imageName"`
}
```

❌ **Mistake 2**: Using `int` instead of `int32`
```go
// Wrong - doesn't match K8s types
Replicas int

// Right
Replicas int32
```

❌ **Mistake 3**: Not documenting fields
```go
// Wrong - no context for users
type DeploymentSpec struct {
    LocalPort int32
}

// Right - explain purpose, constraints, examples
type DeploymentSpec struct {
    // LocalPort is the host machine port for port forwarding
    // Range: 1-65535
    // Example: 8080
    LocalPort int32
}
```

❌ **Mistake 4**: Not using omitempty for optional fields
```go
// Wrong - required fields become nil in JSON
type DeploymentSpec struct {
    KubeContext string  // ← always present, even if empty
}

// Right
type DeploymentSpec struct {
    KubeContext string `yaml:"kubeContext,omitempty"`  // ← only if set
}
```

---

## Next Steps

1. **Create the types** (this task)
2. **Move to Task 1.2** → Write validation rules
3. **Then Task 1.3** → Write config loader
4. Validation will use these types to check values
5. Loader will parse YAML into these types

---

## References

- [Go struct tags](https://golang.org/pkg/reflect/#StructTag)
- [YAML marshaling](https://pkg.go.dev/gopkg.in/yaml.v3)
- [K8s Deployment API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#deploymentspec-v1-apps)
- [K8s EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.30/#envvar-v1-core)

