package portfwd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/nanaki-93/kudev/pkg/logging"
	"github.com/nanaki-93/kudev/pkg/logs"
)

// PortForwarder forwards local ports to Kubernetes pods.
type PortForwarder interface {
	// Forward starts port forwarding in the background.
	// Returns when forwarding is established.
	Forward(ctx context.Context, appName, namespace string, localPort, podPort int32) error

	// Stop terminates port forwarding.
	Stop()
}

// KubernetesPortForwarder implements PortForwarder using client-go.
type KubernetesPortForwarder struct {
	clientset  kubernetes.Interface
	restConfig *rest.Config
	discovery  *logs.PodDiscovery
	logger     logging.LoggerInterface

	// Internal state
	stopChan  chan struct{}
	readyChan chan struct{}
}

// NewKubernetesPortForwarder creates a new port forwarder.
func NewKubernetesPortForwarder(
	clientset kubernetes.Interface,
	restConfig *rest.Config,
	logger logging.LoggerInterface,
) *KubernetesPortForwarder {
	return &KubernetesPortForwarder{
		clientset:  clientset,
		restConfig: restConfig,
		discovery:  logs.NewPodDiscovery(clientset),
		logger:     logger,
	}
}

// Forward starts port forwarding to a pod.
func (pf *KubernetesPortForwarder) Forward(ctx context.Context, appName, namespace string, localPort, podPort int32) error {
	// 1. Check port availability
	if err := checkPortAvailable(localPort); err != nil {
		return fmt.Errorf("port %d is not available: %w\n\nTry a different port with --local-port flag", localPort, err)
	}

	pf.logger.Info("waiting for pod to be ready...",
		"app", appName,
		"namespace", namespace,
	)

	// 2. Wait for a running pod
	pod, err := pf.discovery.DiscoverPod(ctx, appName, namespace, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to find pod: %w", err)
	}

	pf.logger.Info("found pod",
		"pod", pod.Name,
	)

	// 3. Create channels
	pf.stopChan = make(chan struct{}, 1)
	pf.readyChan = make(chan struct{})

	// 4. Build port forward URL
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward", namespace, pod.Name)
	hostURL, err := url.Parse(pf.restConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to parse host URL: %w", err)
	}
	hostURL.Path = path

	// 5. Create SPDY transport
	transport, upgrader, err := spdy.RoundTripperFor(pf.restConfig)
	if err != nil {
		return fmt.Errorf("failed to create transport: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, hostURL)

	// 6. Create port forwarder
	ports := []string{fmt.Sprintf("%d:%d", localPort, podPort)}

	// Use io.Discard for output (we'll log manually)
	fw, err := portforward.New(dialer, ports, pf.stopChan, pf.readyChan, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create port forwarder: %w", err)
	}

	// 7. Start forwarding in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- fw.ForwardPorts()
	}()

	// 8. Wait for ready or error
	select {
	case <-pf.readyChan:
		pf.logger.Info("port forwarding ready",
			"local", fmt.Sprintf("localhost:%d", localPort),
			"pod", fmt.Sprintf("%s:%d", pod.Name, podPort),
		)

		// Start background monitor
		go pf.monitor(ctx, errChan, appName, namespace, localPort, podPort)

		return nil

	case err := <-errChan:
		return fmt.Errorf("port forwarding failed: %w", err)

	case <-ctx.Done():
		pf.Stop()
		return ctx.Err()
	}
}

// monitor watches for errors and attempts reconnection.
func (pf *KubernetesPortForwarder) monitor(ctx context.Context, errChan chan error, appName, namespace string, localPort, podPort int32) {
	for {
		select {
		case <-ctx.Done():
			return

		case err := <-errChan:
			if err != nil {
				pf.logger.Info("port forward disconnected, reconnecting...",
					"error", err,
				)

				// Wait a bit before reconnecting
				time.Sleep(2 * time.Second)

				// Try to reconnect
				if ctx.Err() == nil {
					if err := pf.Forward(ctx, appName, namespace, localPort, podPort); err != nil {
						pf.logger.Error(err, "reconnection failed")
					}
				}
			}
			return
		}
	}
}

// Stop terminates port forwarding.
func (pf *KubernetesPortForwarder) Stop() {
	if pf.stopChan != nil {
		close(pf.stopChan)
	}
}

// checkPortAvailable checks if a local port is available.
func checkPortAvailable(port int32) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

// SuggestAlternativePort finds an available port near the requested one.
func SuggestAlternativePort(preferredPort int32) (int32, error) {
	// Try ports around the preferred one
	for delta := int32(0); delta < 100; delta++ {
		for _, p := range []int32{preferredPort + delta, preferredPort - delta} {
			if p < 1024 || p > 65535 {
				continue
			}
			if checkPortAvailable(p) == nil {
				return p, nil
			}
		}
	}
	return 0, fmt.Errorf("no available ports found near %d", preferredPort)
}

// Ensure KubernetesPortForwarder implements PortForwarder
var _ PortForwarder = (*KubernetesPortForwarder)(nil)
