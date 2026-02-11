package logs

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// PodDiscovery finds pods by label selector.
type PodDiscovery struct {
	clientset kubernetes.Interface
}

// NewPodDiscovery creates a new pod discovery instance.
func NewPodDiscovery(clientset kubernetes.Interface) *PodDiscovery {
	return &PodDiscovery{clientset: clientset}
}

// DiscoverPod finds a pod by app label.
// Waits up to timeout for a pod to exist and be running.
func (pd *PodDiscovery) DiscoverPod(ctx context.Context, appName, namespace string, timeout time.Duration) (*corev1.Pod, error) {
	selector := labels.SelectorFromSet(labels.Set{"app": appName})

	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for pod with label app=%s", appName)
		}

		pods, err := pd.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: selector.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods: %w", err)
		}

		// Find a running pod
		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Status.Phase == corev1.PodRunning {
				return pod, nil
			}
		}

		// Wait and retry
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}
}

// WaitForPodReady waits for a specific pod to be ready.
func (pd *PodDiscovery) WaitForPodReady(ctx context.Context, name, namespace string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for pod %s to be ready", name)
		}

		pod, err := pd.clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod: %w", err)
		}

		if isPodReady(pod) {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}
}

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
