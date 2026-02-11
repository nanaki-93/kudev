// pkg/logs/tailer.go

package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/nanaki-93/kudev/pkg/logging"
)

// LogTailer streams logs from Kubernetes pods.
type LogTailer interface {
	// TailLogs streams logs from the first pod matching the app label.
	TailLogs(ctx context.Context, appName, namespace string) error
}

// KubernetesLogTailer implements LogTailer using client-go.
type KubernetesLogTailer struct {
	clientset kubernetes.Interface
	discovery *PodDiscovery
	logger    logging.LoggerInterface
	output    io.Writer
}

// NewKubernetesLogTailer creates a new log tailer.
func NewKubernetesLogTailer(
	clientset kubernetes.Interface,
	logger logging.LoggerInterface,
	output io.Writer,
) *KubernetesLogTailer {
	return &KubernetesLogTailer{
		clientset: clientset,
		discovery: NewPodDiscovery(clientset),
		logger:    logger,
		output:    output,
	}
}

// TailLogs streams logs from pods with the given app label.
func (lt *KubernetesLogTailer) TailLogs(ctx context.Context, appName, namespace string) error {
	lt.logger.Info("waiting for pods...",
		"app", appName,
		"namespace", namespace,
	)

	// Wait for a running pod
	pod, err := lt.discovery.DiscoverPod(ctx, appName, namespace, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to discover pod: %w", err)
	}

	lt.logger.Info("found pod, streaming logs",
		"pod", pod.Name,
	)

	return lt.streamLogs(ctx, pod.Name, namespace)
}

// streamLogs streams logs from a specific pod.
func (lt *KubernetesLogTailer) streamLogs(ctx context.Context, podName, namespace string) error {
	// Configure log options
	opts := &corev1.PodLogOptions{
		Follow:     true,          // Stream new logs
		TailLines:  int64Ptr(100), // Start with last 100 lines
		Timestamps: true,          // Include timestamps
	}

	// Get log stream
	req := lt.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to open log stream: %w", err)
	}
	defer stream.Close()

	// Stream logs to output
	scanner := bufio.NewScanner(stream)
	// Increase buffer for long log lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fmt.Fprintln(lt.output, scanner.Text())
		}
	}

	if err := scanner.Err(); err != nil {
		// EOF is expected when pod terminates
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("log stream error: %w", err)
	}

	return nil
}

// TailLogsWithRetry streams logs with automatic reconnection on failures.
func (lt *KubernetesLogTailer) TailLogsWithRetry(ctx context.Context, appName, namespace string) error {
	for {
		err := lt.TailLogs(ctx, appName, namespace)

		// Check if we should stop
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			lt.logger.Info("log stream ended, reconnecting...",
				"error", err,
			)
			time.Sleep(2 * time.Second)
			continue
		}

		return nil
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

// Ensure KubernetesLogTailer implements LogTailer
var _ LogTailer = (*KubernetesLogTailer)(nil)
