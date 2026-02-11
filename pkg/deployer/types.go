// pkg/deployer/types.go

package deployer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nanaki-93/kudev/pkg/config"
)

// TemplateData is passed to YAML templates for rendering.
// All fields must match template placeholders exactly.
type TemplateData struct {
	AppName     string
	Namespace   string
	ImageRef    string
	ImageHash   string
	ServicePort int32
	Replicas    int32
	Env         []EnvVar
}

type EnvVar struct {
	Name  string
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
