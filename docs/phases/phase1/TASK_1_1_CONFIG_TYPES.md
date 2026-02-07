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

### Complete Implementation

Create `pkg/config/types.go`:

```go
package config

// DeploymentConfig is the root configuration object.
// It follows K8s API conventions with apiVersion, kind, metadata, and spec.
//
// Example:
//   apiVersion: kudev.io/v1alpha1
//   kind: DeploymentConfig
//   metadata:
//     name: myapp
//   spec:
//     imageName: myapp
//     dockerfilePath: ./Dockerfile
//     namespace: default
//     replicas: 1
//     localPort: 8080
//     servicePort: 8080
type DeploymentConfig struct {
	// APIVersion defines the version of this configuration format.
	// Currently: "kudev.io/v1alpha1"
	// Future versions may have breaking changes.
	APIVersion string `yaml:"apiVersion" json:"apiVersion"`

	// Kind identifies the resource type.
	// Currently: "DeploymentConfig"
	// Allows future extensions like "ServiceConfig", "IngressConfig".
	Kind string `yaml:"kind" json:"kind"`

	// Metadata contains resource identification.
	Metadata ConfigMetadata `yaml:"metadata" json:"metadata"`

	// Spec contains the deployment specification.
	Spec DeploymentSpec `yaml:"spec" json:"spec"`
}

// ConfigMetadata follows K8s naming conventions.
// It identifies the deployed application.
type ConfigMetadata struct {
	// Name is the identifier for this deployment.
	// Used as:
	//   - Kubernetes Deployment name
	//   - Docker image base name
	//   - Service name
	//   - Pod label selector value
	//
	// Requirements (validated in validation.go):
	//   - DNS-1123 compliant: lowercase alphanumeric and hyphens only
	//   - Length: 3-63 characters
	//   - Cannot start or end with hyphen
	//
	// Valid examples:
	//   - "my-app"
	//   - "api"
	//   - "frontend-service"
	//
	// Invalid examples:
	//   - "MyApp" (uppercase)
	//   - "my_app" (underscore)
	//   - "-myapp" (starts with hyphen)
	Name string `yaml:"name" json:"name"`
}

// DeploymentSpec contains all deployment configuration.
type DeploymentSpec struct {
	// ImageName is the container image name (without registry).
	//
	// This is the "short name" of the image that will be built.
	// The registry URL is added during build phase.
	//
	// Examples:
	//   - "myapp" → built as "localhost:5000/myapp:latest"
	//   - "api" → built as "localhost:5000/api:latest"
	//
	// Requirements:
	//   - Lowercase alphanumeric and hyphens
	//   - Should match metadata.name in most cases
	ImageName string `yaml:"imageName" json:"imageName"`

	// DockerfilePath is the path to the Dockerfile relative to project root.
	//
	// Discovery algorithm:
	//   1. If absolute path: use as-is
	//   2. If relative: resolve from project root
	//   3. If file doesn't exist: validation will fail
	//
	// Examples:
	//   - "./Dockerfile" (standard)
	//   - "./docker/Dockerfile.dev" (multi-stage)
	//   - "Dockerfile" (in project root)
	//
	// Project root detection:
	//   - Directory containing .git
	//   - Directory containing go.mod
	//   - Directory containing .kudev.yaml
	DockerfilePath string `yaml:"dockerfilePath" json:"dockerfilePath"`

	// Namespace is the target Kubernetes namespace.
	//
	// This is where the Deployment, Service, and Pods will be created.
	// Multi-namespace support is planned for Phase 3.
	//
	// Default: "default" (if not specified)
	// Common values:
	//   - "default" (development)
	//   - "dev" (development namespace)
	//   - "stage" (staging namespace)
	//
	// Requirements:
	//   - DNS-1123 compliant (same as metadata.name)
	//   - Must exist in K8s cluster (or be created by you)
	//
	// Safety: Kudev won't create namespaces automatically.
	// Create with: kubectl create namespace my-namespace
	Namespace string `yaml:"namespace" json:"namespace"`

	// Replicas is the number of pod replicas to create.
	//
	// Type: int32 (matches K8s Deployment.spec.replicas)
	// Default: 1 (if not specified or 0)
	// Minimum: 1 (validated)
	// Maximum: system-dependent (usually 1000+)
	//
	// Examples:
	//   - 1: single pod (typical for development)
	//   - 3: three pods (test HA locally)
	//   - 5: stress test
	//
	// Change at runtime with:
	//   kubectl scale deployment myapp --replicas=3
	Replicas int32 `yaml:"replicas" json:"replicas"`

	// LocalPort is the host machine port for port forwarding.
	//
	// This is the port you access from your browser:
	//   - http://localhost:8080
	//   - http://127.0.0.1:8080
	//
	// Used by: kudev portfwd → kubectl port-forward pod :localPort
	//
	// Range: 1-65535 (validated)
	// Common values:
	//   - 8080: HTTP services
	//   - 5432: PostgreSQL
	//   - 27017: MongoDB
	//   - 6379: Redis
	//
	// Conflict resolution:
	//   - If port already in use: kudev will show clear error
	//   - Change port in .kudev.yaml and retry
	//
	// Note: Requires elevated permissions (sudo) for ports < 1024
	LocalPort int32 `yaml:"localPort" json:"localPort"`

	// ServicePort is the container port inside the pod.
	//
	// This is the port your application listens on inside the container.
	// Should match EXPOSE in your Dockerfile.
	//
	// Example Dockerfile:
	//   FROM node:18
	//   EXPOSE 3000
	//   CMD ["npm", "start"]
	//
	// Then in .kudev.yaml:
	//   servicePort: 3000
	//   localPort: 3000  (or different)
	//
	// Typical values:
	//   - 8080: Default for Go/Java services
	//   - 3000: Node.js services
	//   - 5000: Python Flask
	//   - 8000: Python Django
	//
	// Used by K8s Service to forward traffic:
	//   Service:8080 → Pod:servicePort
	ServicePort int32 `yaml:"servicePort" json:"servicePort"`

	// Env is a list of environment variables for the container.
	//
	// These are injected into the Kubernetes Pod spec.
	// Used for configuration that changes by deployment environment.
	//
	// Example:
	//   env:
	//     - name: LOG_LEVEL
	//       value: "info"
	//     - name: DEBUG
	//       value: "true"
	//     - name: DATABASE_URL
	//       value: "postgres://postgres:5432/mydb"
	//
	// Notes:
	//   - Values are ALWAYS strings (converted from YAML)
	//   - For secrets: use K8s Secrets (future enhancement)
	//   - Order doesn't matter
	//   - Duplicate names: last one wins (validated)
	//
	// ValueFrom (ConfigMaps, Secrets) - NOT YET SUPPORTED
	// Phase 4 will add support for:
	//   - ConfigMap references
	//   - Secret references
	//   - Field references (pod name, namespace, etc.)
	Env []EnvVar `yaml:"env" json:"env"`

	// KubeContext is the optional Kubernetes context to use.
	//
	// If specified:
	//   - Overrides context whitelist validation
	//   - Forces use of this context
	//   - Fails if context doesn't exist in kubeconfig
	//
	// If NOT specified:
	//   - Uses context whitelist (docker-desktop, minikube, kind-*)
	//   - Requires --force-context to use others
	//
	// Use case 1: Development environment pinning
	//   kubeContext: docker-desktop
	//   # Forces this config to always use Docker Desktop
	//   # Team members always deploy to same cluster
	//
	// Use case 2: CI/CD pinning
	//   kubeContext: kind-ci
	//   # CI pipeline always uses this cluster
	//
	// Safety check flow:
	//   1. Load current context from kubeconfig
	//   2. If kubeContext set: require exact match or fail
	//   3. If kubeContext not set: check whitelist
	//
	// Omitted: empty string, ignored
	KubeContext string `yaml:"kubeContext" json:"kubeContext,omitempty"`

	// BuildContextExclusions is a list of paths to exclude from Docker build.
	//
	// These paths are COPY'ed into the image during build:
	//   COPY . /app
	//
	// Excluding unnecessary files:
	//   - Speeds up builds (small context)
	//   - Reduces image size (less data copied)
	//   - Prevents secrets being included
	//
	// Default exclusions (always applied):
	//   - .git/
	//   - .gitignore
	//   - .dockerignore
	//   - .kudev.yaml
	//   - node_modules/ (if exists)
	//   - vendor/ (Go)
	//
	// Additional exclusions specified here:
	//   buildContextExclusions:
	//     - .env
	//     - __pycache__
	//     - .pytest_cache
	//     - coverage/
	//     - build/
	//
	// Paths can be:
	//   - File: ".env"
	//   - Directory: "vendor/" or "vendor"
	//   - Glob: "*.log" (NOT YET SUPPORTED - Phase 2)
	//
	// Note: .dockerignore is the real mechanism
	// Kudev generates .dockerignore from this list
	BuildContextExclusions []string `yaml:"buildContextExclusions" json:"buildContextExclusions,omitempty"`
}

// EnvVar represents a single environment variable.
//
// Follows K8s v1.EnvVar structure (same as Pod spec).
// Used by: Kubernetes deployment manifest generation (Phase 3).
//
// Example:
//   - name: LOG_LEVEL
//     value: info
//   - name: APP_DEBUG
//     value: "true"
//
// Notes:
//   - Name: Must be valid environment variable name (alphanumeric, _, uppercase)
//   - Value: Always string, can contain any characters including spaces
//   - YAML tip: Use quotes for values like "true", "false", "123"
type EnvVar struct {
	// Name is the environment variable name.
	//
	// Requirements:
	//   - Valid shell variable name: [A-Z0-9_]+
	//   - Typically UPPERCASE
	//   - No spaces or special characters
	//
	// Common names:
	//   - LOG_LEVEL
	//   - DATABASE_URL
	//   - API_KEY
	//   - PORT
	//
	// Invalid names (validated in Phase 1.2):
	//   - "my-var" (hyphens not allowed)
	//   - "123var" (starts with number)
	Name string `yaml:"name" json:"name"`

	// Value is the environment variable value.
	//
	// Always a string in YAML/JSON.
	// Your application is responsible for parsing types:
	//   - "3000" → parseInt("3000") = 3000
	//   - "true" → parseBool("true") = true
	//   - "[]" → parseJSON("[]") = []
	//
	// YAML quirk - Always quote non-string values:
	//   env:
	//     - name: PORT
	//       value: "3000"    # ← must be quoted
	//     - name: DEBUG
	//       value: "true"    # ← must be quoted
	//     - name: URL
	//       value: http://localhost:8080  # ← can be unquoted
	//
	// Future enhancement (Phase 4):
	//   Will support valueFrom:
	//     valueFrom:
	//       configMapKeyRef:
	//         name: myconfig
	//         key: log_level
	Value string `yaml:"value" json:"value"`
}

// DeploymentConfigFactory returns a configuration with K8s API defaults.
// Used primarily for testing and initialization.
func NewDeploymentConfig(name string) *DeploymentConfig {
	return &DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: ConfigMetadata{
			Name: name,
		},
		Spec: DeploymentSpec{
			ImageName:   name,
			Namespace:   "default",
			Replicas:    1,
			LocalPort:   8080,
			ServicePort: 8080,
		},
	}
}
```

---

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

