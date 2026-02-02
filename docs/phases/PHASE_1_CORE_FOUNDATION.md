# Phase 1: Core Foundation (CLI & Config)

**Objective**: Create a tool that can "speak" to the user and read their intent. Establish the CLI structure, configuration schema, and context awareness to prevent accidental deployments.

**Timeline**: 1-2 weeks  
**Difficulty**: 🟢 Beginner-Friendly (core Go patterns, no K8s complexity yet)  
**Dependencies**: None (foundation layer)

---

## 📋 Architecture Overview

This phase establishes the foundational components:

```
┌────────────────────────────────────────┐
│        User (Terminal Input)           │
└────────────────┬───────────────────────┘
                 │
         ┌───────▼────────┐
         │   Cobra (CLI)  │
         └───────┬────────┘
                 │
    ┌────────────┼────────────┐
    ▼            ▼            ▼
┌─────────┐ ┌─────────┐ ┌─────────┐
│ Config  │ │Validator│ │ Logger  │
│ Loader  │ │(Context)│ │ (Klog)  │
└─────────┘ └─────────┘ └─────────┘
    │            │            │
    └────────────┼────────────┘
                 │
         ┌───────▼────────┐
         │  Ready for     │
         │  Phase 2 (Build)
         └────────────────┘
```

---

## 🎯 Core Decisions

### Decision 1.1: Configuration File Format

**Question**: What format should users use to specify deployment intent?

| Format | Pros | Cons |
|--------|------|------|
| YAML | Human-friendly, industry standard | Indentation errors, learning curve |
| TOML | Structured, clear syntax | Less common in K8s ecosystem |
| JSON | Programmatic, structured | Not human-friendly |

**🎯 Decision**: **YAML** (`.kudev.yaml`)
- Industry standard in K8s ecosystem
- Matches kubectl conventions
- Human-readable and editableformat

**Schema Location**: `.kudev.yaml` in project root

**Validation Strategy**:
```go
// Validate on load using K8s patterns
// - Required fields: imageName, dockerfilePath, namespace
// - Type checking: replicas must be int32, ports must be valid
// - DNS validation: namespace/appname must follow DNS-1123
```

### Decision 1.2: CLI Framework

**Question**: Which CLI framework to use?

| Framework | Pros | Cons |
|-----------|------|------|
| Cobra | K8s standard, rich features, wide adoption | Verbose |
| Urfave/CLI | Simple, lightweight | Less powerful |
| Flag package | Minimal dependencies | Limited features |

**🎯 Decision**: **Cobra** (by spf13)
- Used by kubectl, Docker, Kubernetes ecosystem
- Standard plugins follow this pattern
- Rich subcommand support

**Command Structure**:
```
kudev
├── version          # Print version
├── init             # Initialize .kudev.yaml
├── validate         # Validate config
├── up               # Build + deploy
├── down             # Delete deployment
├── status           # Show deployment status
├── logs             # Stream pod logs
├── portfwd          # Port forwarding
├── watch            # Watch for changes
└── debug            # Debug info
```

### Decision 1.3: Context Safety Strategy

**Question**: How do we prevent deploying to production?

| Strategy | Pros | Cons |
|----------|------|------|
| Whitelist contexts | Safe by default | Restrictive for advanced users |
| Blacklist contexts | Flexible | Easy to accidentally bypass |
| Config pinning | Very explicit | Requires per-project setup |

**🎯 Decision**: **Whitelist contexts** (with `--force-context` override)
- Fail safely by default
- Explicit allowlist: `docker-desktop`, `minikube`, `kind-*`
- Require `--force-context` flag for non-whitelisted contexts (with warning)

**Validation Flow**:
```
1. Load kubeconfig
2. Get current context
3. Check context against whitelist
4. If not in whitelist:
   - If --force-context flag: warn but continue
   - Else: fail with clear error message listing allowed contexts
```

---

## 📝 Detailed Tasks

### Task 1.1: Define Configuration Types

**Goal**: Create Go types that match the YAML configuration schema.

**Files to Create**:
- `pkg/config/types.go` — Configuration structs matching K8s patterns

**Configuration Schema**:
```yaml
apiVersion: kudev.io/v1alpha1
kind: DeploymentConfig
metadata:
  name: myapp
spec:
  # Container image name (without registry)
  imageName: myapp
  
  # Dockerfile location relative to project root
  dockerfilePath: ./Dockerfile
  
  # Target Kubernetes namespace
  namespace: default
  
  # Number of replicas
  replicas: 1
  
  # Local port for port forwarding
  localPort: 8080
  
  # Container port
  servicePort: 8080
  
  # Environment variables
  env:
    - name: LOG_LEVEL
      value: info
    - name: DEBUG
      value: "false"

  # Optional: Kubeconfig context to pin
  kubeContext: docker-desktop
  
  # Optional: Files to exclude from build hash
  buildContextExclusions:
    - .git
    - node_modules
    - vendor
```

**Implementation Hints**:

```go
// pkg/config/types.go

// DeploymentConfig is the root configuration
type DeploymentConfig struct {
    APIVersion string            `yaml:"apiVersion"`
    Kind       string            `yaml:"kind"`
    Metadata   ConfigMetadata    `yaml:"metadata"`
    Spec       DeploymentSpec    `yaml:"spec"`
}

// ConfigMetadata matches K8s naming conventions
type ConfigMetadata struct {
    Name string `yaml:"name"`
}

// DeploymentSpec contains deployment configuration
type DeploymentSpec struct {
    ImageName                 string          `yaml:"imageName"`
    DockerfilePath            string          `yaml:"dockerfilePath"`
    Namespace                 string          `yaml:"namespace"`
    Replicas                  int32           `yaml:"replicas"`
    LocalPort                 int32           `yaml:"localPort"`
    ServicePort               int32           `yaml:"servicePort"`
    Env                       []EnvVar        `yaml:"env"`
    KubeContext               string          `yaml:"kubeContext,omitempty"`
    BuildContextExclusions    []string        `yaml:"buildContextExclusions,omitempty"`
}

// EnvVar represents an environment variable
type EnvVar struct {
    Name  string `yaml:"name"`
    Value string `yaml:"value"`
}
```

**Success Criteria**:
- ✅ Types marshal/unmarshal to YAML correctly
- ✅ JSON tags present for JSON output (future use)
- ✅ Comments explain each field
- ✅ Follows K8s API conventions (APIVersion, Kind, metadata)

**Hints for Implementation**:
- Use `yaml.Unmarshal()` from `sigs.k8s.io/yaml` (K8s standard)
- Add validation tags for clarity
- Use `omitempty` for optional fields
- Follow naming: `apiVersion` not `api_version`

---

### Task 1.2: Implement Configuration Validation

**Goal**: Validate config values and provide helpful error messages.

**Files to Create**:
- `pkg/config/validation.go` — Validation rules
- `pkg/config/errors.go` — Validation error types

**Validation Rules**:

```go
// Validate performs full configuration validation
func (c *DeploymentConfig) Validate() error {
    // Required fields
    if c.Metadata.Name == "" {
        return fmt.Errorf("metadata.name is required")
    }
    
    if c.Spec.ImageName == "" {
        return fmt.Errorf("spec.imageName is required")
    }
    
    if c.Spec.DockerfilePath == "" {
        return fmt.Errorf("spec.dockerfilePath is required")
    }
    
    // Validate naming (DNS-1123 compliant)
    if !isValidDNSName(c.Metadata.Name) {
        return fmt.Errorf("metadata.name must be DNS-1123 compliant (lowercase, hyphens only)")
    }
    
    // Validate ports
    if c.Spec.LocalPort < 1 || c.Spec.LocalPort > 65535 {
        return fmt.Errorf("spec.localPort must be between 1 and 65535")
    }
    
    if c.Spec.ServicePort < 1 || c.Spec.ServicePort > 65535 {
        return fmt.Errorf("spec.servicePort must be between 1 and 65535")
    }
    
    // Validate replicas
    if c.Spec.Replicas < 1 {
        return fmt.Errorf("spec.replicas must be at least 1")
    }
    
    // Validate namespace
    if c.Spec.Namespace == "" {
        return fmt.Errorf("spec.namespace is required")
    }
    
    if !isValidDNSName(c.Spec.Namespace) {
        return fmt.Errorf("spec.namespace must be DNS-1123 compliant")
    }
    
    return nil
}
```

**Success Criteria**:
- ✅ Validates all required fields
- ✅ Validates numeric ranges (ports 1-65535, replicas > 0)
- ✅ Validates DNS names (DNS-1123 compliant)
- ✅ Error messages are specific and actionable
- ✅ No validation errors on valid config

**Hints for Implementation**:
- DNS-1123 pattern: `^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
- Use regex for DNS validation or K8s validation utils
- Collect all errors and report them together (not just first error)

---

### Task 1.3: Implement Configuration Loader

**Goal**: Load config from `.kudev.yaml` with intelligent discovery.

**Files to Create**:
- `pkg/config/loader.go` — ConfigLoader interface and implementation
- `pkg/config/defaults.go` — Default values

**ConfigLoader Interface**:

```go
// ConfigLoader loads and saves configuration
type ConfigLoader interface {
    // Load discovers and loads .kudev.yaml
    Load(ctx context.Context) (*DeploymentConfig, error)
    
    // Save writes configuration to file
    Save(ctx context.Context, cfg *DeploymentConfig) error
}
```

**Discovery Algorithm**:

```
1. Check for --config flag (override)
2. Check current directory for .kudev.yaml
3. Check parent directories up to project root (heuristics: .git, go.mod, etc.)
4. Check home directory ~/.kudev/config
5. If not found: return helpful error

Return: first found config or error
```

**Default Values**:

```go
// ApplyDefaults fills in missing values
func (c *DeploymentConfig) ApplyDefaults() {
    if c.Spec.Namespace == "" {
        c.Spec.Namespace = "default"
    }
    
    if c.Spec.Replicas == 0 {
        c.Spec.Replicas = 1
    }
    
    if c.Spec.LocalPort == 0 {
        c.Spec.LocalPort = 8080
    }
    
    if c.Spec.ServicePort == 0 {
        c.Spec.ServicePort = 8080
    }
}
```

**Success Criteria**:
- ✅ Discovers .kudev.yaml in current or parent directories
- ✅ Respects --config flag override
- ✅ Applies sensible defaults
- ✅ Validates config after loading
- ✅ Clear error if config not found

**Hints for Implementation**:
- Use `filepath.Walk()` to search parent directories
- Use Viper for config loading (K8s standard)
- Wrap errors with context: `fmt.Errorf("failed to load config: %w", err)`
- Support environment variable substitution with `${VAR_NAME}` pattern

---

### Task 1.4: Implement Kubeconfig Reader & Context Validation

**Goal**: Load kubeconfig and validate K8s context safety.

**Files to Create**:
- `pkg/kubeconfig/context.go` — Kubeconfig reading and context detection
- `pkg/kubeconfig/validator.go` — Context validation logic

**Kubeconfig Loading**:

```go
// LoadCurrentContext reads kubeconfig and returns current context
func LoadCurrentContext() (*KubeconfigContext, error) {
    kubeconfig := os.Getenv("KUBECONFIG")
    if kubeconfig == "" {
        kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config")
    }
    
    // Use client-go patterns (same as kubectl)
    config, err := clientcmd.LoadFromFile(kubeconfig)
    if err != nil {
        return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
    }
    
    currentContext := config.CurrentContext
    // Parse context details
    // Return context info
}
```

**Context Validation**:

```go
// ContextValidator checks if context is safe
type ContextValidator struct {
    allowedContexts []string  // Whitelist
    forceContext    bool      // --force-context override
}

// Validate checks context safety
func (cv *ContextValidator) Validate(currentContext string) error {
    // Check if context matches whitelist
    for _, allowed := range cv.allowedContexts {
        if isMatchPattern(currentContext, allowed) {
            return nil  // Valid
        }
    }
    
    // Not in whitelist
    if cv.forceContext {
        logger.Warn("context '%s' not in whitelist, proceeding with --force-context", currentContext)
        return nil
    }
    
    return fmt.Errorf(
        "context '%s' not allowed for safety\nAllowed contexts: %v\nUse --force-context to override",
        currentContext,
        cv.allowedContexts,
    )
}
```

**Allowed Contexts** (hardcoded in code, can be overridden in config):
```go
defaultAllowedContexts := []string{
    "docker-desktop",
    "docker-for-desktop",
    "minikube",
    "kind-*",           // Regex pattern
}
```

**Success Criteria**:
- ✅ Reads kubeconfig from standard locations
- ✅ Detects current context
- ✅ Validates against whitelist
- ✅ `--force-context` override works
- ✅ Helpful error message with allowed contexts

**Hints for Implementation**:
- Use `client-go/tools/clientcmd` (same as kubectl)
- Support regex patterns for kind clusters (`kind-.*`)
- Use `filepath.Match()` or `regexp` for pattern matching
- Provide clear error with list of allowed contexts

---

### Task 1.5: Implement Cobra CLI Scaffolding

**Goal**: Build CLI structure with all commands defined.

**Files to Create**:
- `cmd/main.go` — Entry point
- `cmd/root.go` — Root command definition
- `cmd/version.go` — Version command
- `cmd/init.go` — Init command
- `cmd/validate.go` — Validate command
- Additional command files (to be filled in Phase 2+)

**Root Command Structure**:

```go
// cmd/root.go

var rootCmd = &cobra.Command{
    Use:   "kudev",
    Short: "Kubernetes development helper",
    Long: `Kudev streamlines local Kubernetes development with automatic
building, deploying, and live-reloading.`,
    
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        // Global initialization here
        // - Load kubeconfig
        // - Load config
        // - Setup logging
        return nil
    },
}

func init() {
    // Global flags
    rootCmd.PersistentFlags().String("config", "", "Path to .kudev.yaml")
    rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging")
    rootCmd.PersistentFlags().Bool("force-context", false, "Skip context safety check")
    
    // Add subcommands
    rootCmd.AddCommand(versionCmd)
    rootCmd.AddCommand(initCmd)
    rootCmd.AddCommand(validateCmd)
    // ... more commands
}
```

**Version Command**:

```go
// cmd/version.go

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("kudev version %s\n", version.Version)
    },
}
```

**Init Command** (generates `.kudev.yaml`):

```go
// cmd/init.go

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize .kudev.yaml",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Ask user for app name
        // 2. Ask for dockerfile path
        // 3. Create default DeploymentConfig
        // 4. Apply defaults
        // 5. Save to .kudev.yaml
        // 6. Validate
        return nil
    },
}
```

**Validate Command**:

```go
// cmd/validate.go

var validateCmd = &cobra.Command{
    Use:   "validate",
    Short: "Validate .kudev.yaml",
    RunE: func(cmd *cobra.Command, args []string) error {
        // 1. Load config
        // 2. Validate
        // 3. Print success message
        return nil
    },
}
```

**Success Criteria**:
- ✅ All commands parse correctly
- ✅ Help text displays (`kudev --help`, `kudev init --help`)
- ✅ Global flags work (--config, --debug, --force-context)
- ✅ Errors propagate to root for uniform handling
- ✅ Version command works

**Hints for Implementation**:
- Use `cobra generate` to scaffold commands
- Put minimal logic in cmd/ files; delegate to pkg/
- Use `cmd.RunE` (error-returning) not `cmd.Run`
- Use `PersistentFlags` for global flags, `Flags` for command-specific

---

### Task 1.6: Implement Logging with Klog

**Goal**: Setup structured logging compatible with K8s patterns.

**Files to Create**:
- `pkg/logging/logger.go` — Logger initialization

**Logger Setup**:

```go
// pkg/logging/logger.go

var logger klog.Logger

// Init initializes the logger based on flags
func Init(debug bool) {
    if debug {
        klog.SetLogger(klog.NewKlogr())
        // Set verbosity level
        flag.Set("v", "4")  // V(4) for debug logs
    } else {
        klog.SetLogger(klog.NewKlogr())
        flag.Set("v", "0")  // V(0) for normal
    }
}

// Get returns the configured logger
func Get() klog.Logger {
    return klog.Background()
}
```

**Usage in Code**:

```go
logger := logging.Get()

// Info logs
logger.Info("deployment created", "namespace", ns, "name", appName)

// Error logs
logger.Error(err, "failed to deploy", "namespace", ns)

// Debug logs (only printed with --debug)
logger.V(4).Info("processing file", "path", filepath)
```

**Success Criteria**:
- ✅ Logging respects --debug flag
- ✅ Info and Error levels work
- ✅ Debug logs only appear with --debug
- ✅ Structured logging with key-value pairs
- ✅ Timestamps included in output

**Hints for Implementation**:
- Use `klog/v2` (K8s standard)
- Initialize in `cmd/root.go` before running commands
- Use `logger.V(level).Info()` for verbosity-based logs
- Avoid print statements; use logger instead

---

## 🧪 Testing Strategy for Phase 1

### Unit Tests

**Test Files to Create**:
- `pkg/config/types_test.go` — Type marshaling
- `pkg/config/validation_test.go` — Validation rules
- `pkg/config/loader_test.go` — Config loading
- `pkg/kubeconfig/validator_test.go` — Context validation
- `cmd/root_test.go` — CLI parsing

**Test Pattern** (table-driven):

```go
// pkg/config/validation_test.go

func TestValidate(t *testing.T) {
    tests := []struct {
        name    string
        config  *DeploymentConfig
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid config",
            config: &DeploymentConfig{
                Metadata: ConfigMetadata{Name: "app"},
                Spec: DeploymentSpec{
                    ImageName:      "app",
                    DockerfilePath: "./Dockerfile",
                    Namespace:      "default",
                },
            },
            wantErr: false,
        },
        {
            name: "missing app name",
            config: &DeploymentConfig{
                Metadata: ConfigMetadata{Name: ""},
                Spec:     DeploymentSpec{},
            },
            wantErr: true,
            errMsg:  "metadata.name is required",
        },
        {
            name: "invalid port",
            config: &DeploymentConfig{
                Metadata: ConfigMetadata{Name: "app"},
                Spec: DeploymentSpec{
                    LocalPort: 70000,  // Out of range
                },
            },
            wantErr: true,
            errMsg:  "localPort must be between 1 and 65535",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
            if err != nil && tt.errMsg != "" {
                if !strings.Contains(err.Error(), tt.errMsg) {
                    t.Errorf("Validate() error message %q does not contain %q", err.Error(), tt.errMsg)
                }
            }
        })
    }
}
```

**Test Coverage Targets**:
- Config validation: 85%+
- Config loader: 80%+
- Context validator: 80%+
- CLI parsing: 75%+

### No Integration Tests Yet
- Don't need K8s cluster for Phase 1
- All tests use mocks/fakes

---

## ✅ Phase 1 Success Criteria

- ✅ Config types defined and match YAML schema
- ✅ Config loads from `.kudev.yaml` with discovery
- ✅ Config validates with specific error messages
- ✅ All CLI commands respond (no errors)
- ✅ Context validation prevents unsafe deployments
- ✅ Help text is clear and actionable
- ✅ Logging works with --debug flag
- ✅ Unit tests >80% coverage
- ✅ No external cluster required for testing

---

## ⚠️ Critical Issues & Mitigations

| Issue | Mitigation | Priority |
|-------|-----------|----------|
| Config discovery finds wrong file | Explicit search order; allow --config override | High |
| Context validator regex too loose | Test with edge cases; require explicit opt-in | High |
| Logging too verbose | Support log levels; document klog flags | Medium |
| Error messages unclear | Include examples in output ("Did you mean...?") | Medium |
| Config validation incomplete | Test with invalid YAML; add fuzzing | Medium |

---

## 🎓 Learning Resources

- [Cobra Framework](https://cobra.dev/) — Official docs
- [Viper Configuration](https://github.com/spf13/viper) — Config examples
- [Client-Go Kubeconfig Loading](https://github.com/kubernetes/client-go/blob/master/tools/clientcmd/loader.go) — Reference implementation
- [Klog Usage](https://kubernetes.io/docs/concepts/cluster-administration/manage-deployment/#debug-logging) — K8s logging

---

**Next**: [Phase 2 - Image Pipeline](./PHASE_2_IMAGE_PIPELINE.md) 🔨
