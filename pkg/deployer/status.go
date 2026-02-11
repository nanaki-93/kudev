package deployer

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// Status returns the current deployment status.
func (kd *KubernetesDeployer) Status(ctx context.Context, appName, namespace string) (*DeploymentStatus, error) {
	kd.logger.Debug("getting deployment status",
		"app", appName,
		"namespace", namespace,
	)

	// Get deployment
	deployment, err := kd.clientset.AppsV1().Deployments(namespace).Get(
		ctx, appName, metav1.GetOptions{},
	)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("deployment not found: %s/%s", namespace, appName)
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get pods by label selector
	selector := labels.SelectorFromSet(labels.Set{"app": appName})
	pods, err := kd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Determine desired replicas
	var desiredReplicas int32 = 1
	if deployment.Spec.Replicas != nil {
		desiredReplicas = *deployment.Spec.Replicas
	}

	// Build pod statuses
	podStatuses := buildPodStatuses(pods)

	// Determine overall status
	statusCode := computeStatusCode(deployment.Status.ReadyReplicas, desiredReplicas, podStatuses)

	// Get image hash from labels
	imageHash := ""
	if deployment.Labels != nil {
		imageHash = deployment.Labels["kudev-hash"]
	}

	status := &DeploymentStatus{
		DeploymentName:  deployment.Name,
		Namespace:       deployment.Namespace,
		ReadyReplicas:   deployment.Status.ReadyReplicas,
		DesiredReplicas: desiredReplicas,
		Status:          statusCode.String(),
		Pods:            podStatuses,
		Message:         buildStatusMessage(statusCode, deployment.Status.ReadyReplicas, desiredReplicas),
		ImageHash:       imageHash,
		LastUpdated:     time.Now(),
	}

	return status, nil
}

// buildPodStatuses converts K8s pod list to our PodStatus slice.
func buildPodStatuses(pods *corev1.PodList) []PodStatus {
	var statuses []PodStatus

	for _, pod := range pods.Items {
		status := PodStatus{
			Name:      pod.Name,
			Status:    string(pod.Status.Phase),
			Ready:     isPodReady(&pod),
			CreatedAt: pod.CreationTimestamp.Time,
		}

		// Count container restarts
		for _, cs := range pod.Status.ContainerStatuses {
			status.Restarts += cs.RestartCount

			// Get waiting/terminated message
			if cs.State.Waiting != nil && cs.State.Waiting.Message != "" {
				status.Message = cs.State.Waiting.Message
			}
			if cs.State.Terminated != nil && cs.State.Terminated.Message != "" {
				status.Message = cs.State.Terminated.Message
			}
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// isPodReady checks if all containers in pod are ready.
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// computeStatusCode determines overall deployment health.
func computeStatusCode(ready, desired int32, pods []PodStatus) StatusCode {
	if desired == 0 {
		return StatusUnknown
	}

	if ready >= desired {
		return StatusRunning
	}

	if ready == 0 {
		// Check for crash loops
		for _, pod := range pods {
			if pod.Restarts > 3 {
				return StatusFailed
			}
		}
		return StatusPending
	}

	return StatusDegraded
}

// buildStatusMessage creates a user-friendly status message.
func buildStatusMessage(status StatusCode, ready, desired int32) string {
	switch status {
	case StatusRunning:
		return fmt.Sprintf("All %d replicas are running", desired)
	case StatusPending:
		return fmt.Sprintf("Waiting for pods to start (0/%d ready)", desired)
	case StatusDegraded:
		return fmt.Sprintf("Partially running (%d/%d ready)", ready, desired)
	case StatusFailed:
		return "Pods are failing - check logs with 'kudev logs'"
	default:
		return "Unable to determine status"
	}
}

// WaitForReady waits until deployment is ready or timeout.
func (kd *KubernetesDeployer) WaitForReady(ctx context.Context, appName, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for deployment to be ready")
		}

		status, err := kd.Status(ctx, appName, namespace)
		if err != nil {
			// Deployment might not exist yet
			kd.logger.Debug("waiting for deployment", "error", err)
		} else if status.IsReady() {
			kd.logger.Info("deployment is ready",
				"app", appName,
				"replicas", status.ReadyReplicas,
			)
			return nil
		} else {
			kd.logger.Debug("waiting for deployment",
				"app", appName,
				"ready", status.ReadyReplicas,
				"desired", status.DesiredReplicas,
			)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}
}
