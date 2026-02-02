# Phase 3: Manifest Orchestration (Deployment)

**Objective**: Take the built image and deploy it to the K8s cluster using embedded templates and dynamic templating. Implement create/update logic for safe deployments.

**Timeline**: 1-2 weeks  
**Difficulty**: 🟡 Intermediate (client-go, templates, K8s API patterns)  
**Dependencies**: Phase 1-2 (Config, Logger, Builder, Image loading)

---

## 📋 Architecture Overview

This phase handles K8s API interactions:

```
┌─────────────────────────────────────────┐
│       Phase 2: Built ImageRef           │
└────────────────┬────────────────────────┘
                 │
    ┌────────────▼───────────┐
    │  Embed YAML Templates  │
    │  (deployment.yaml)     │
    │  (service.yaml)        │
    └────────────┬───────────┘
                 │
    ┌────────────▼───────────┐
    │  Render Templates      │
    │  with TemplateData     │
    │  (image, env, etc)     │
    └────────────┬───────────┘
                 │
    ┌────────────▼───────────┐
    │  Deployer Interface    │
    │  - Upsert (create/upd) │
    │  - Delete (safe)       │
    │  - Status              │
    └────────────┬───────────┘
                 │
    ┌────────────▼───────────┐
    │  K8s API (client-go)   │
    │  - AppsV1             │
    │  - CoreV1             │
    └────────────┬───────────┘
                 │
         ┌───────▼────────┐
         │  Cluster Ready │
         │  for Phase 4   │
         └────────────────┘
```

---

## 🎯 Core Decisions

### Decision 3.1: Manifest Management Strategy

**Question**: How should users define K8s manifests?

| Strategy | Pros | Cons |
|----------|------|------|
| Embedded templates | Simple, no files, works out-of-box | Less flexible |
| User-provided YAML | Flexible, standard K8s | Requires file management |
| Hybrid | Embedded defaults + override option | Complex |

**🎯 Decision**: **Embedded templates** (Phase 1) → Allow overrides in Phase 3b
- Reduces boilerplate and entry barriers
- Single source of truth in binary
- Users can inspect generated manifests with `--dry-run`

### Decision 3.2: Template Engine

**Question**: How to inject values into YAML?

| Engine | Pros | Cons |
|--------|------|------|
| Go `text/template` | Built-in, simple | Limited features |
| Kustomize | Industry standard | Heavy dependency |
| Helm | Powerful | Overkill for simple use case |

**🎯 Decision**: **Go `text/template`**
- Built into standard library
- Sufficient for simple substitution (image, env, ports)
- Easy to understand and debug
- Minimal dependencies

### Decision 3.3: Create vs Update Logic

**Question**: How to handle existing deployments?

| Approach | Pros | Cons |
|----------|------|------|
| Always recreate | Simple | Downtime, pod eviction |
| Get→Modify→Update | Safe, preserves state | Risky if not careful |
| Three-way merge | Most correct | Complex |

**🎯 Decision**: **Get→Modify→Update (strategic)**
- Check if Deployment exists
- If exists: Get current, modify only image/env, Update
- If not: Create both Deployment and Service
- Use labels (`managed-by: kudev`) to track our resources

---

## 📝 Detailed Tasks

### Task 3.1: Create Embedded YAML Templates

**Goal**: Define base templates for Deployment and Service.

**Files to Create**:
- `templates/deployment.yaml` — Deployment template
- `templates/service.yaml` — Service template

**Deployment Template**:

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
        env:
        {{ range .Env }}
        - name: {{ .Name }}
          value: {{ .Value | quote }}
        {{ end }}
        imagePullPolicy: IfNotPresent
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
          requests:
            cpu: "250m"
            memory: "256Mi"
```

**Service Template**:

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

**Critical Labels**:
- `managed-by: kudev` — Identifies resources created by kudev
- `kudev-hash: {HASH}` — Tracks deployed source code version
- `app: {APPNAME}` — Standard K8s label for pod selection

**Success Criteria**:
- ✅ Templates are valid YAML with placeholders
- ✅ All labels present for identification
- ✅ `imagePullPolicy: IfNotPresent` ensures local images work
- ✅ Service selector matches Deployment pod labels

**Hints for Implementation**:
- Use `//go:embed` to embed templates in binary
- Support template functions like `quote` for string values
- Add resource limits (prevents resource hogging)
- Document all template variables

---

### Task 3.2: Create Template Data Structures

**Goal**: Define data passed to templates.

**Files to Create**:
- `pkg/deployer/types.go` — TemplateData and related types

**TemplateData Structure**:

```go
// pkg/deployer/types.go

// TemplateData is passed to YAML templates
type TemplateData struct {
    AppName     string
    Namespace   string
    ImageRef    string      // e.g., myapp:kudev-a1b2c3d4
    ImageHash   string      // Source code hash
    ServicePort int32
    Replicas    int32
    Env         []EnvVar
}

// EnvVar represents environment variable
type EnvVar struct {
    Name  string
    Value string
}

// DeploymentStatus represents deployment state
type DeploymentStatus struct {
    DeploymentName  string
    Namespace       string
    ReadyReplicas   int32
    DesiredReplicas int32
    Status          string  // "Running", "Pending", "Failed", "CrashLoopBackOff"
    Pods            []PodStatus
    Message         string  // Helpful status message
}

// PodStatus represents individual pod state
type PodStatus struct {
    Name      string
    Status    string
    Restarts  int32
    CreatedAt time.Time
}

// Deployer interface
type Deployer interface {
    // Upsert creates or updates deployment
    Upsert(ctx context.Context, opts DeploymentOptions) (*DeploymentStatus, error)
    
    // Delete removes deployment and service
    Delete(ctx context.Context, appName, namespace string) error
    
    // Status returns current deployment status
    Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error)
}

// DeploymentOptions contains input for deployment
type DeploymentOptions struct {
    Config    *config.DeploymentConfig  // From .kudev.yaml
    ImageRef  string                     // Built image reference
    ImageHash string                     // Source code hash
}
```

**Success Criteria**:
- ✅ All template variables represented
- ✅ TemplateData is complete and unambiguous
- ✅ Status types capture K8s state
- ✅ Deployer interface is minimal

---

### Task 3.3: Implement Template Rendering

**Goal**: Parse YAML templates and inject values.

**Files to Create**:
- `pkg/deployer/renderer.go` — Template rendering logic

**Renderer Implementation**:

```go
// pkg/deployer/renderer.go

type Renderer struct {
    deploymentTpl string  // Embedded template content
    serviceTpl    string  // Embedded template content
}

func NewRenderer(deploymentTpl, serviceTpl string) *Renderer {
    return &Renderer{
        deploymentTpl: deploymentTpl,
        serviceTpl:    serviceTpl,
    }
}

// RenderDeployment renders Deployment YAML
func (r *Renderer) RenderDeployment(data TemplateData) (*appsv1.Deployment, error) {
    // Parse template
    tpl, err := template.New("deployment").Parse(r.deploymentTpl)
    if err != nil {
        return nil, fmt.Errorf("failed to parse deployment template: %w", err)
    }
    
    // Execute template
    var buf bytes.Buffer
    if err := tpl.Execute(&buf, data); err != nil {
        return nil, fmt.Errorf("failed to render deployment template: %w", err)
    }
    
    // Unmarshal into K8s object
    deployment := &appsv1.Deployment{}
    if err := yaml.Unmarshal(buf.Bytes(), deployment); err != nil {
        return nil, fmt.Errorf("invalid deployment YAML: %w", err)
    }
    
    return deployment, nil
}

// RenderService renders Service YAML
func (r *Renderer) RenderService(data TemplateData) (*corev1.Service, error) {
    tpl, err := template.New("service").Parse(r.serviceTpl)
    if err != nil {
        return nil, fmt.Errorf("failed to parse service template: %w", err)
    }
    
    var buf bytes.Buffer
    if err := tpl.Execute(&buf, data); err != nil {
        return nil, fmt.Errorf("failed to render service template: %w", err)
    }
    
    service := &corev1.Service{}
    if err := yaml.Unmarshal(buf.Bytes(), service); err != nil {
        return nil, fmt.Errorf("invalid service YAML: %w", err)
    }
    
    return service, nil
}
```

**Success Criteria**:
- ✅ Templates render without errors
- ✅ Invalid template syntax detected early
- ✅ Missing variables cause clear errors
- ✅ Output is valid K8s YAML
- ✅ Supports all template functions (quote, etc)

**Hints for Implementation**:
- Use `k8s.io/apimachinery` for K8s types
- Test with invalid YAML; verify error messages
- Use `sigs.k8s.io/yaml` for YAML parsing (K8s standard)

---

### Task 3.4: Implement Deployer Interface (Upsert Logic)

**Goal**: Create/update deployments safely.

**Files to Create**:
- `pkg/deployer/deployer.go` — Deployer implementation
- `pkg/deployer/upsert.go` — Upsert logic

**Deployer Implementation**:

```go
// pkg/deployer/deployer.go

type KubernetesDeployer struct {
    clientset *kubernetes.Clientset
    renderer  *Renderer
    logger    logging.Logger
}

func NewKubernetesDeployer(
    clientset *kubernetes.Clientset,
    renderer *Renderer,
    logger logging.Logger,
) *KubernetesDeployer {
    return &KubernetesDeployer{
        clientset: clientset,
        renderer:  renderer,
        logger:    logger,
    }
}

// Upsert creates or updates deployment
func (kd *KubernetesDeployer) Upsert(ctx context.Context, opts DeploymentOptions) (*DeploymentStatus, error) {
    // 1. Prepare template data
    data := TemplateData{
        AppName:     opts.Config.Metadata.Name,
        Namespace:   opts.Config.Spec.Namespace,
        ImageRef:    opts.ImageRef,
        ImageHash:   opts.ImageHash,
        ServicePort: opts.Config.Spec.ServicePort,
        Replicas:    opts.Config.Spec.Replicas,
        Env:         opts.Config.Spec.Env,
    }
    
    // 2. Render manifests
    deployment, err := kd.renderer.RenderDeployment(data)
    if err != nil {
        return nil, fmt.Errorf("failed to render deployment: %w", err)
    }
    
    service, err := kd.renderer.RenderService(data)
    if err != nil {
        return nil, fmt.Errorf("failed to render service: %w", err)
    }
    
    // 3. Create namespace if needed
    if err := kd.ensureNamespace(ctx, opts.Config.Spec.Namespace); err != nil {
        return nil, fmt.Errorf("failed to create namespace: %w", err)
    }
    
    // 4. Upsert Deployment
    if err := kd.upsertDeployment(ctx, deployment); err != nil {
        return nil, fmt.Errorf("failed to upsert deployment: %w", err)
    }
    
    // 5. Upsert Service
    if err := kd.upsertService(ctx, service); err != nil {
        return nil, fmt.Errorf("failed to upsert service: %w", err)
    }
    
    // 6. Return status
    return kd.Status(ctx, opts.Config.Metadata.Name, opts.Config.Spec.Namespace)
}

// upsertDeployment creates or updates deployment
func (kd *KubernetesDeployer) upsertDeployment(ctx context.Context, dep *appsv1.Deployment) error {
    deployments := kd.clientset.AppsV1().Deployments(dep.Namespace)
    
    // Try to get existing
    existing, err := deployments.Get(ctx, dep.Name, metav1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            // Create new
            _, err := deployments.Create(ctx, dep, metav1.CreateOptions{})
            if err != nil {
                return fmt.Errorf("failed to create deployment: %w", err)
            }
            kd.logger.Info("deployment created", "name", dep.Name, "namespace", dep.Namespace)
            return nil
        }
        return fmt.Errorf("failed to get deployment: %w", err)
    }
    
    // Update existing: merge important fields
    existing.Spec.Replicas = dep.Spec.Replicas
    existing.Spec.Template.Spec.Containers[0].Image = dep.Spec.Template.Spec.Containers[0].Image
    existing.Spec.Template.Spec.Containers[0].Env = dep.Spec.Template.Spec.Containers[0].Env
    
    // Preserve existing labels, add kudev labels
    if existing.Labels == nil {
        existing.Labels = make(map[string]string)
    }
    existing.Labels["kudev-hash"] = dep.Labels["kudev-hash"]
    
    _, err = deployments.Update(ctx, existing, metav1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update deployment: %w", err)
    }
    
    kd.logger.Info("deployment updated", "name", dep.Name, "namespace", dep.Namespace)
    return nil
}

// upsertService creates or updates service
func (kd *KubernetesDeployer) upsertService(ctx context.Context, svc *corev1.Service) error {
    services := kd.clientset.CoreV1().Services(svc.Namespace)
    
    existing, err := services.Get(ctx, svc.Name, metav1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            _, err := services.Create(ctx, svc, metav1.CreateOptions{})
            if err != nil {
                return fmt.Errorf("failed to create service: %w", err)
            }
            kd.logger.Info("service created", "name", svc.Name, "namespace", svc.Namespace)
            return nil
        }
        return fmt.Errorf("failed to get service: %w", err)
    }
    
    // Update service: preserve clusterIP, update ports and selectors
    existing.Spec.Ports = svc.Spec.Ports
    existing.Spec.Selector = svc.Spec.Selector
    
    _, err = services.Update(ctx, existing, metav1.UpdateOptions{})
    if err != nil {
        return fmt.Errorf("failed to update service: %w", err)
    }
    
    kd.logger.Info("service updated", "name", svc.Name, "namespace", svc.Namespace)
    return nil
}

// ensureNamespace creates namespace if not exists
func (kd *KubernetesDeployer) ensureNamespace(ctx context.Context, namespace string) error {
    if namespace == "default" {
        return nil  // default always exists
    }
    
    ns, err := kd.clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
    if err == nil {
        return nil  // exists
    }
    
    if !errors.IsNotFound(err) {
        return err
    }
    
    // Create it
    ns = &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{
            Name: namespace,
            Labels: map[string]string{
                "managed-by": "kudev",
            },
        },
    }
    
    _, err = kd.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
    if err != nil {
        return fmt.Errorf("failed to create namespace: %w", err)
    }
    
    kd.logger.Info("namespace created", "name", namespace)
    return nil
}
```

**Success Criteria**:
- ✅ Creates new deployments
- ✅ Updates existing deployments (image, env only)
- ✅ Preserves existing pod labels/annotations
- ✅ Creates service alongside deployment
- ✅ Creates namespace if needed
- ✅ Proper error handling with K8s error types

**Hints for Implementation**:
- Use `errors.IsNotFound()` to detect missing resources
- Use strategic merge patch for updates (not full replacement)
- Preserve clusterIP when updating services
- Document label contracts

---

### Task 3.5: Implement Status Retrieval

**Goal**: Query deployment status.

**Files to Create**:
- `pkg/deployer/status.go` — Status query logic

**Status Implementation**:

```go
// pkg/deployer/status.go

// Status returns current deployment status
func (kd *KubernetesDeployer) Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error) {
    // Get deployment
    deployment, err := kd.clientset.AppsV1().Deployments(namespace).Get(ctx, appName, metav1.GetOptions{})
    if err != nil {
        if errors.IsNotFound(err) {
            return nil, fmt.Errorf("deployment not found: %s/%s", namespace, appName)
        }
        return nil, fmt.Errorf("failed to get deployment: %w", err)
    }
    
    // Get pods
    selector := labels.SelectorFromSet(labels.Set{"app": appName})
    pods, err := kd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: selector.String(),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to list pods: %w", err)
    }
    
    // Build status
    status := &DeploymentStatus{
        DeploymentName:  deployment.Name,
        Namespace:       deployment.Namespace,
        ReadyReplicas:   deployment.Status.ReadyReplicas,
        DesiredReplicas: *deployment.Spec.Replicas,
        Status:          getDeploymentStatusString(deployment),
        Pods:            buildPodStatuses(pods),
    }
    
    return status, nil
}

func getDeploymentStatusString(dep *appsv1.Deployment) string {
    if dep.Status.ReadyReplicas == *dep.Spec.Replicas {
        return "Running"
    }
    
    if dep.Status.Replicas == 0 {
        return "Pending"
    }
    
    if dep.Status.UnavailableReplicas > 0 {
        return "Degraded"
    }
    
    return "Unknown"
}

func buildPodStatuses(pods *corev1.PodList) []PodStatus {
    var statuses []PodStatus
    
    for _, pod := range pods.Items {
        status := PodStatus{
            Name:      pod.Name,
            Status:    string(pod.Status.Phase),
            CreatedAt: pod.CreationTimestamp.Time,
        }
        
        // Count restarts
        for _, container := range pod.Status.ContainerStatuses {
            status.Restarts += container.RestartCount
        }
        
        statuses = append(statuses, status)
    }
    
    return statuses
}
```

**Success Criteria**:
- ✅ Returns accurate replica counts
- ✅ Lists pods with statuses
- ✅ Handles missing deployments gracefully
- ✅ Status strings are meaningful
- ✅ Tracks restart counts

---

### Task 3.6: Implement Safe Delete with Labels

**Goal**: Delete deployment using labels to avoid accidents.

**Files to Create**:
- `pkg/deployer/delete.go` — Safe deletion

**Delete Implementation**:

```go
// pkg/deployer/delete.go

// Delete removes deployment and service
func (kd *KubernetesDeployer) Delete(ctx context.Context, appName, namespace string) error {
    // Delete deployment
    deployments := kd.clientset.AppsV1().Deployments(namespace)
    if err := deployments.Delete(ctx, appName, metav1.DeleteOptions{
        PropagationPolicy: metav1.DeletePropagationForeground,  // Wait for pods to terminate
    }); err != nil && !errors.IsNotFound(err) {
        return fmt.Errorf("failed to delete deployment: %w", err)
    }
    
    kd.logger.Info("deployment deleted", "name", appName, "namespace", namespace)
    
    // Delete service
    services := kd.clientset.CoreV1().Services(namespace)
    if err := services.Delete(ctx, appName, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
        return fmt.Errorf("failed to delete service: %w", err)
    }
    
    kd.logger.Info("service deleted", "name", appName, "namespace", namespace)
    
    return nil
}
```

**Success Criteria**:
- ✅ Deletes only resources with matching labels
- ✅ Idempotent (safe to run multiple times)
- ✅ Cascading delete removes pods
- ✅ Clear log messages

---

## 🧪 Testing Strategy for Phase 3

### Unit Tests with Fake Client

**Test Files to Create**:
- `pkg/deployer/deployer_test.go` — Main deployer logic
- `pkg/deployer/renderer_test.go` — Template rendering
- `pkg/deployer/status_test.go` — Status queries

**Fake Client Pattern**:

```go
// pkg/deployer/deployer_test.go

func TestUpsertCreatesDeployment(t *testing.T) {
    // Create fake clientset
    fakeClientset := fake.NewSimpleClientset()
    
    renderer := NewRenderer(deploymentTpl, serviceTpl)
    deployer := NewKubernetesDeployer(fakeClientset, renderer, logger)
    
    opts := DeploymentOptions{
        Config: &config.DeploymentConfig{...},
        ImageRef: "myapp:kudev-a1b2c3d4",
        ImageHash: "a1b2c3d4",
    }
    
    status, err := deployer.Upsert(context.Background(), opts)
    
    // Verify deployment was created
    if err != nil {
        t.Fatalf("Upsert failed: %v", err)
    }
    
    deployment, err := fakeClientset.AppsV1().Deployments("default").Get(
        context.Background(), "app", metav1.GetOptions{},
    )
    if err != nil {
        t.Fatalf("deployment not found: %v", err)
    }
    
    if deployment.Name != "app" {
        t.Errorf("expected app, got %s", deployment.Name)
    }
}
```

**Test Coverage Targets**:
- Deployer: 85%+
- Renderer: 90%+
- Status: 80%+

---

## ✅ Phase 3 Success Criteria

- ✅ Templates embedded and render correctly
- ✅ Deployer creates new deployments
- ✅ Deployer updates existing deployments
- ✅ Service created and selects pods
- ✅ Delete targets only kudev resources
- ✅ Status queries accurate
- ✅ Unit tests >80% coverage

---

**Next**: [Phase 4 - Developer Experience](./PHASE_4_DEVELOPER_EXPERIENCE.md) 🎯
