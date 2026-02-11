package config

// DeploymentConfig is the root configuration object.
// It follows K8s API conventions with apiVersion, kind, metadata, and spec.
// Example:
//
//	apiVersion: kudev.io/v1alpha1
//	kind: DeploymentConfig
//	metadata:
//	  name: myapp
//	spec:
//	  imageName: myapp
//	  dockerfilePath: ./Dockerfile
//	  namespace: default
//	  replicas: 1
//	  localPort: 8080
//	  servicePort: 8080
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
	Metadata MetadataConfig `yaml:"metadata" json:"metadata"`
	// Spec contains the deployment specification.
	Spec SpecConfig `yaml:"spec" json:"spec"`

	ProjectRoot string `yaml:"-" json:"-"`
}

// MetadataConfig follows K8s naming conventions.
// It identifies the deployed application.
type MetadataConfig struct {
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

// SpecConfig contains all deployment configuration.
type SpecConfig struct {
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

	// DockerfilePath is the savePath to the Dockerfile relative to project root.
	//
	// Discovery algorithm:
	//   1. If absolute savePath: use as-is
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
	Value string `yaml:"value" json:"value,omitempty"`
}

// NewDeploymentConfig returns a configuration with K8s API defaults.
// Used primarily for testing and initialization.
func NewDeploymentConfig(appName string) *DeploymentConfig {
	return &DeploymentConfig{
		APIVersion: "kudev.io/v1alpha1",
		Kind:       "DeploymentConfig",
		Metadata: MetadataConfig{
			Name: appName,
		},
		Spec: SpecConfig{
			ImageName:      appName,
			DockerfilePath: "./Dockerfile",
			Namespace:      "default",
			Replicas:       1,
			LocalPort:      8080,
			ServicePort:    8080,
			Env:            []EnvVar{},
		},
	}
}
