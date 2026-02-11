// pkg/deployer/deployer_test.go

package deployer

import (
	"context"
	"testing"

	"github.com/nanaki-93/kudev/test/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/nanaki-93/kudev/pkg/config"
	"github.com/nanaki-93/kudev/templates"
)

func TestUpsert_CreateNew(t *testing.T) {
	// Create fake clientset (empty cluster)
	fakeClient := fake.NewSimpleClientset()

	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	opts := DeploymentOptions{
		Config: &config.DeploymentConfig{
			Metadata: config.MetadataConfig{Name: "test-app"},
			Spec: config.SpecConfig{
				Namespace:   "default",
				Replicas:    2,
				ServicePort: 8080,
			},
		},
		ImageRef:  "test-app:kudev-12345678",
		ImageHash: "12345678",
	}

	status, err := deployer.Upsert(context.Background(), opts)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify deployment was created
	deployment, err := fakeClient.AppsV1().Deployments("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("deployment not found: %v", err)
	}

	if deployment.Name != "test-app" {
		t.Errorf("name = %q, want %q", deployment.Name, "test-app")
	}

	// Verify service was created
	service, err := fakeClient.CoreV1().Services("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("service not found: %v", err)
	}

	if service.Name != "test-app" {
		t.Errorf("service name = %q, want %q", service.Name, "test-app")
	}

	if status == nil {
		t.Error("status is nil")
	}
}

func TestUpsert_UpdateExisting(t *testing.T) {
	// Create fake clientset with existing deployment
	existingDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Labels: map[string]string{
				"app":        "test-app",
				"managed-by": "kudev",
				"kudev-hash": "old-hash",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "test-app"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-app",
							Image: "test-app:old-image",
						},
					},
				},
			},
		},
	}

	existingService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.100", // Existing ClusterIP
			Ports: []corev1.ServicePort{
				{Port: 8080},
			},
			Selector: map[string]string{"app": "test-app"},
		},
	}

	fakeClient := fake.NewSimpleClientset(existingDeployment, existingService)

	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	opts := DeploymentOptions{
		Config: &config.DeploymentConfig{
			Metadata: config.MetadataConfig{Name: "test-app"},
			Spec: config.SpecConfig{
				Namespace:   "default",
				Replicas:    3, // Changed!
				ServicePort: 8080,
			},
		},
		ImageRef:  "test-app:kudev-new-hash", // Changed!
		ImageHash: "new-hash",
	}

	_, err := deployer.Upsert(context.Background(), opts)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify deployment was updated
	deployment, _ := fakeClient.AppsV1().Deployments("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)

	if *deployment.Spec.Replicas != 3 {
		t.Errorf("replicas = %d, want 3", *deployment.Spec.Replicas)
	}

	if deployment.Spec.Template.Spec.Containers[0].Image != "test-app:kudev-new-hash" {
		t.Errorf("image not updated")
	}

	if deployment.Labels["kudev-hash"] != "new-hash" {
		t.Errorf("hash label not updated")
	}

	// Verify ClusterIP was preserved
	service, _ := fakeClient.CoreV1().Services("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)

	if service.Spec.ClusterIP != "10.0.0.100" {
		t.Errorf("ClusterIP changed! was 10.0.0.100, now %s", service.Spec.ClusterIP)
	}
}

func TestUpsert_CreatesNamespace(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	opts := DeploymentOptions{
		Config: &config.DeploymentConfig{
			Metadata: config.MetadataConfig{Name: "test-app"},
			Spec: config.SpecConfig{
				Namespace:   "custom-ns", // Non-default namespace
				Replicas:    1,
				ServicePort: 8080,
			},
		},
		ImageRef:  "test-app:latest",
		ImageHash: "12345678",
	}

	_, err := deployer.Upsert(context.Background(), opts)
	if err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify namespace was created
	ns, err := fakeClient.CoreV1().Namespaces().Get(
		context.Background(), "custom-ns", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("namespace not created: %v", err)
	}

	if ns.Labels["managed-by"] != "kudev" {
		t.Error("namespace missing managed-by label")
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

// Add to pkg/deployer/deployer_test.go

func TestStatus_DeploymentExists(t *testing.T) {
	// Create fake deployment with status
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
			Labels: map[string]string{
				"app":        "test-app",
				"kudev-hash": "abc12345",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test-app"},
			},
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 2,
			Replicas:      2,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-abc123",
			Namespace: "default",
			Labels:    map[string]string{"app": "test-app"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{RestartCount: 0},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment, pod)

	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	status, err := deployer.Status(context.Background(), "test-app", "default")
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	if status.DeploymentName != "test-app" {
		t.Errorf("name = %q, want %q", status.DeploymentName, "test-app")
	}

	if status.ReadyReplicas != 2 {
		t.Errorf("ready = %d, want 2", status.ReadyReplicas)
	}

	if status.Status != "Running" {
		t.Errorf("status = %q, want %q", status.Status, "Running")
	}

	if status.ImageHash != "abc12345" {
		t.Errorf("hash = %q, want %q", status.ImageHash, "abc12345")
	}

	if !status.IsReady() {
		t.Error("IsReady() should be true")
	}
}

func TestStatus_DeploymentNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()

	renderer, _ := NewRenderer(
		templates.DeploymentTemplate,
		templates.ServiceTemplate,
	)

	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	_, err := deployer.Status(context.Background(), "nonexistent", "default")
	if err == nil {
		t.Error("expected error for nonexistent deployment")
	}
}

func TestStatus_Degraded(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1, // Only 1 of 3 ready
			Replicas:      3,
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment)
	renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	status, _ := deployer.Status(context.Background(), "test-app", "default")

	if status.Status != "Degraded" {
		t.Errorf("status = %q, want %q", status.Status, "Degraded")
	}
}

func TestComputeStatusCode(t *testing.T) {
	tests := []struct {
		name     string
		ready    int32
		desired  int32
		pods     []PodStatus
		expected StatusCode
	}{
		{"all ready", 3, 3, nil, StatusRunning},
		{"more than desired", 4, 3, nil, StatusRunning},
		{"some ready", 1, 3, nil, StatusDegraded},
		{"none ready", 0, 3, nil, StatusPending},
		{"crash loop", 0, 3, []PodStatus{{Restarts: 10}}, StatusFailed},
		{"zero desired", 0, 0, nil, StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeStatusCode(tt.ready, tt.desired, tt.pods)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

// Add to pkg/deployer/deployer_test.go

func TestDelete_ExistingResources(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment, service)
	renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	err := deployer.Delete(context.Background(), "test-app", "default")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deployment was deleted
	_, err = fakeClient.AppsV1().Deployments("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)
	if !errors.IsNotFound(err) {
		t.Error("deployment should be deleted")
	}

	// Verify service was deleted
	_, err = fakeClient.CoreV1().Services("default").Get(
		context.Background(), "test-app", metav1.GetOptions{},
	)
	if !errors.IsNotFound(err) {
		t.Error("service should be deleted")
	}
}

func TestDelete_Idempotent(t *testing.T) {
	// Empty cluster - nothing to delete
	fakeClient := fake.NewSimpleClientset()
	renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	// Should not error even if resources don't exist
	err := deployer.Delete(context.Background(), "nonexistent", "default")
	if err != nil {
		t.Errorf("Delete should be idempotent, got: %v", err)
	}

	// Call again - still no error
	err = deployer.Delete(context.Background(), "nonexistent", "default")
	if err != nil {
		t.Errorf("Delete should be idempotent on second call, got: %v", err)
	}
}

func TestDelete_PartialResources(t *testing.T) {
	// Only deployment exists, no service
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewSimpleClientset(deployment)
	renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	err := deployer.Delete(context.Background(), "test-app", "default")
	if err != nil {
		t.Errorf("Should handle partial resources, got: %v", err)
	}
}

func TestDeleteByLabels(t *testing.T) {
	// Create multiple kudev-managed resources
	dep1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app1",
			Namespace: "default",
			Labels:    map[string]string{"managed-by": "kudev"},
		},
	}
	dep2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app2",
			Namespace: "default",
			Labels:    map[string]string{"managed-by": "kudev"},
		},
	}
	// This one is NOT managed by kudev
	dep3 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-app",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewSimpleClientset(dep1, dep2, dep3)
	renderer, _ := NewRenderer(templates.DeploymentTemplate, templates.ServiceTemplate)
	deployer := NewKubernetesDeployer(fakeClient, renderer, &util.MockLogger{})

	err := deployer.DeleteByLabels(context.Background(), "default")
	if err != nil {
		t.Fatalf("DeleteByLabels failed: %v", err)
	}

	// other-app should still exist (not managed by kudev)
	_, err = fakeClient.AppsV1().Deployments("default").Get(
		context.Background(), "other-app", metav1.GetOptions{},
	)
	if errors.IsNotFound(err) {
		t.Error("other-app should NOT be deleted")
	}
}
