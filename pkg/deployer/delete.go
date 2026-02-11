// pkg/deployer/delete.go

package deployer

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete removes the deployment and associated service.
// Only deletes resources with matching name.
// Safe to call multiple times (idempotent).
func (kd *KubernetesDeployer) Delete(ctx context.Context, appName, namespace string) error {
	kd.logger.Info("deleting deployment",
		"app", appName,
		"namespace", namespace,
	)

	var deleteErrors []string

	// Delete Deployment
	if err := kd.deleteDeployment(ctx, appName, namespace); err != nil {
		deleteErrors = append(deleteErrors, fmt.Sprintf("deployment: %v", err))
	}

	// Delete Service
	if err := kd.deleteService(ctx, appName, namespace); err != nil {
		deleteErrors = append(deleteErrors, fmt.Sprintf("service: %v", err))
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("deletion errors: %v", deleteErrors)
	}

	kd.logger.Info("deletion completed",
		"app", appName,
		"namespace", namespace,
	)

	return nil
}

// deleteDeployment removes a Deployment.
func (kd *KubernetesDeployer) deleteDeployment(ctx context.Context, name, namespace string) error {
	deployments := kd.clientset.AppsV1().Deployments(namespace)

	// Use Foreground deletion to wait for pods to terminate
	propagation := metav1.DeletePropagationForeground

	err := deployments.Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})

	if err != nil {
		if errors.IsNotFound(err) {
			kd.logger.Debug("deployment already deleted",
				"name", name,
				"namespace", namespace,
			)
			return nil // Idempotent
		}
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	kd.logger.Info("deployment deleted",
		"name", name,
		"namespace", namespace,
	)

	return nil
}

// deleteService removes a Service.
func (kd *KubernetesDeployer) deleteService(ctx context.Context, name, namespace string) error {
	services := kd.clientset.CoreV1().Services(namespace)

	err := services.Delete(ctx, name, metav1.DeleteOptions{})

	if err != nil {
		if errors.IsNotFound(err) {
			kd.logger.Debug("service already deleted",
				"name", name,
				"namespace", namespace,
			)
			return nil // Idempotent
		}
		return fmt.Errorf("failed to delete service: %w", err)
	}

	kd.logger.Info("service deleted",
		"name", name,
		"namespace", namespace,
	)

	return nil
}

// DeleteByLabels removes all resources with the kudev managed-by label.
// Useful for cleanup of orphaned resources.
func (kd *KubernetesDeployer) DeleteByLabels(ctx context.Context, namespace string) error {
	kd.logger.Info("deleting all kudev resources",
		"namespace", namespace,
	)

	labelSelector := "managed-by=kudev"

	// Delete Deployments
	deployments := kd.clientset.AppsV1().Deployments(namespace)
	if err := deployments.DeleteCollection(ctx,
		metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: labelSelector},
	); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete deployments: %w", err)
	}

	// Delete Services
	services := kd.clientset.CoreV1().Services(namespace)
	serviceList, err := services.List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	for _, svc := range serviceList.Items {
		if err := services.Delete(ctx, svc.Name, metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete service %s: %w", svc.Name, err)
		}
		kd.logger.Info("service deleted", "name", svc.Name)
	}

	kd.logger.Info("all kudev resources deleted",
		"namespace", namespace,
	)

	return nil
}

// WaitForDeletion waits until deployment is fully deleted.
func (kd *KubernetesDeployer) WaitForDeletion(ctx context.Context, appName, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for deletion")
		}

		_, err := kd.clientset.AppsV1().Deployments(namespace).Get(
			ctx, appName, metav1.GetOptions{},
		)

		if errors.IsNotFound(err) {
			kd.logger.Info("deployment fully deleted",
				"app", appName,
				"namespace", namespace,
			)
			return nil
		}

		if err != nil {
			return fmt.Errorf("error checking deployment: %w", err)
		}

		kd.logger.Debug("waiting for deletion",
			"app", appName,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}
}
