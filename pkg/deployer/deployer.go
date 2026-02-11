// pkg/deployer/deployer.go

package deployer

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// KubernetesDeployer implements Deployer using client-go.
type KubernetesDeployer struct {
	clientset kubernetes.Interface
	renderer  *Renderer
	logger    logging.LoggerInterface
}

// NewKubernetesDeployer creates a new deployer.
func NewKubernetesDeployer(
	clientset kubernetes.Interface,
	renderer *Renderer,
	logger logging.LoggerInterface,
) *KubernetesDeployer {
	return &KubernetesDeployer{
		clientset: clientset,
		renderer:  renderer,
		logger:    logger,
	}
}

// Upsert creates or updates deployment and service.
func (kd *KubernetesDeployer) Upsert(ctx context.Context, opts DeploymentOptions) (*DeploymentStatus, error) {
	// 1. Prepare template data
	data := NewTemplateData(opts)

	kd.logger.Info("starting deployment",
		"app", data.AppName,
		"namespace", data.Namespace,
		"image", data.ImageRef,
	)

	// 2. Render manifests
	deployment, err := kd.renderer.RenderDeployment(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render deployment: %w", err)
	}

	service, err := kd.renderer.RenderService(data)
	if err != nil {
		return nil, fmt.Errorf("failed to render service: %w", err)
	}

	// 3. Ensure namespace exists
	if err := kd.ensureNamespace(ctx, data.Namespace); err != nil {
		return nil, fmt.Errorf("failed to ensure namespace: %w", err)
	}

	// 4. Upsert Deployment
	if err := kd.upsertDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("failed to upsert deployment: %w", err)
	}

	// 5. Upsert Service
	if err := kd.upsertService(ctx, service); err != nil {
		return nil, fmt.Errorf("failed to upsert service: %w", err)
	}

	kd.logger.Info("deployment completed successfully",
		"app", data.AppName,
		"namespace", data.Namespace,
	)

	// 6. Return current status
	return kd.Status(ctx, data.AppName, data.Namespace)
}

// upsertDeployment creates or updates a Deployment.
func (kd *KubernetesDeployer) upsertDeployment(ctx context.Context, desired *appsv1.Deployment) error {
	deployments := kd.clientset.AppsV1().Deployments(desired.Namespace)

	// Try to get existing
	existing, err := deployments.Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new deployment
			_, err := deployments.Create(ctx, desired, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create deployment: %w", err)
			}
			kd.logger.Info("deployment created",
				"name", desired.Name,
				"namespace", desired.Namespace,
			)
			return nil
		}
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Update existing deployment
	// Preserve fields that shouldn't change
	existing.Spec.Replicas = desired.Spec.Replicas

	// Update container image and env
	if len(existing.Spec.Template.Spec.Containers) > 0 &&
		len(desired.Spec.Template.Spec.Containers) > 0 {
		existing.Spec.Template.Spec.Containers[0].Image =
			desired.Spec.Template.Spec.Containers[0].Image
		existing.Spec.Template.Spec.Containers[0].Env =
			desired.Spec.Template.Spec.Containers[0].Env
	}

	// Update kudev labels
	if existing.Labels == nil {
		existing.Labels = make(map[string]string)
	}
	existing.Labels["kudev-hash"] = desired.Labels["kudev-hash"]

	// Update pod template labels
	if existing.Spec.Template.Labels == nil {
		existing.Spec.Template.Labels = make(map[string]string)
	}
	existing.Spec.Template.Labels["managed-by"] = "kudev"

	_, err = deployments.Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	kd.logger.Info("deployment updated",
		"name", desired.Name,
		"namespace", desired.Namespace,
	)

	return nil
}

// upsertService creates or updates a Service.
func (kd *KubernetesDeployer) upsertService(ctx context.Context, desired *corev1.Service) error {
	services := kd.clientset.CoreV1().Services(desired.Namespace)

	existing, err := services.Get(ctx, desired.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new service
			_, err := services.Create(ctx, desired, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}
			kd.logger.Info("service created",
				"name", desired.Name,
				"namespace", desired.Namespace,
			)
			return nil
		}
		return fmt.Errorf("failed to get service: %w", err)
	}

	// Update existing service
	// CRITICAL: Preserve ClusterIP (cannot be changed)
	desired.Spec.ClusterIP = existing.Spec.ClusterIP
	desired.Spec.ClusterIPs = existing.Spec.ClusterIPs

	// Copy resource version for update
	desired.ResourceVersion = existing.ResourceVersion

	_, err = services.Update(ctx, desired, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	kd.logger.Info("service updated",
		"name", desired.Name,
		"namespace", desired.Namespace,
	)

	return nil
}

// ensureNamespace creates namespace if it doesn't exist.
func (kd *KubernetesDeployer) ensureNamespace(ctx context.Context, namespace string) error {
	// Skip for default namespace
	if namespace == "default" {
		return nil
	}

	namespaces := kd.clientset.CoreV1().Namespaces()

	_, err := namespaces.Get(ctx, namespace, metav1.GetOptions{})
	if err == nil {
		// Namespace exists
		return nil
	}

	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check namespace: %w", err)
	}

	// Create namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"managed-by": "kudev",
			},
		},
	}

	_, err = namespaces.Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		// Ignore AlreadyExists (race condition)
		if errors.IsAlreadyExists(err) {
			return nil
		}
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	kd.logger.Info("namespace created", "name", namespace)
	return nil
}

// Ensure KubernetesDeployer implements Deployer
var _ Deployer = (*KubernetesDeployer)(nil)
