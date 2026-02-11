// pkg/deployer/types_test.go

package deployer

import (
	"testing"

	"github.com/nanaki-93/kudev/pkg/config"
)

func TestNewTemplateData(t *testing.T) {
	cfg := &config.DeploymentConfig{
		Metadata: config.MetadataConfig{
			Name: "myapp",
		},
		Spec: config.SpecConfig{
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
		name   string
		status DeploymentStatus
		want   bool
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
