package errors

import "fmt"

// Config errors

func ConfigNotFound(path string) *ConfigError {
	return &ConfigError{
		Message:    "Configuration file not found: " + path,
		Suggestion: "Run 'kudev init' to create a new configuration, or specify path with --config",
	}
}

func ConfigInvalid(reason string, cause error) *ConfigError {
	return &ConfigError{
		Message:    "Invalid configuration: " + reason,
		Suggestion: "Check your .kudev.yaml file for syntax errors",
		Cause:      cause,
	}
}

func ConfigMissingField(field string) *ConfigError {
	return &ConfigError{
		Message:    "Missing required field: " + field,
		Suggestion: "Add '" + field + "' to your .kudev.yaml configuration",
	}
}

// Kubernetes auth errors

func KubeconfigNotFound() *KubeAuthError {
	return &KubeAuthError{
		Message:    "Kubeconfig file not found",
		Suggestion: "Set KUBECONFIG environment variable or create ~/.kube/config",
	}
}

func KubeContextNotFound(context string) *KubeAuthError {
	return &KubeAuthError{
		Message:    "Kubernetes context not found: " + context,
		Suggestion: "Run 'kubectl config get-contexts' to see available contexts",
	}
}

func KubeContextNotAllowed(context string) *KubeAuthError {
	return &KubeAuthError{
		Message:    "Context '" + context + "' is not allowed for local development",
		Suggestion: "Use a local cluster like Docker Desktop, Minikube, or Kind",
	}
}

func KubeConnectionFailed(cause error) *KubeAuthError {
	return &KubeAuthError{
		Message:    "Failed to connect to Kubernetes cluster",
		Suggestion: "Ensure your cluster is running and kubectl is configured correctly",
		Cause:      cause,
	}
}

// Build errors

func DockerNotRunning(cause error) *BuildError {
	return &BuildError{
		Message:    "Docker daemon is not running",
		Suggestion: "Start Docker Desktop or run 'sudo systemctl start docker'",
		Cause:      cause,
	}
}

func DockerBuildFailed(cause error) *BuildError {
	return &BuildError{
		Message:    "Docker build failed",
		Suggestion: "Check the build output above for errors in your Dockerfile",
		Cause:      cause,
	}
}

func DockerfileNotFound(path string) *BuildError {
	return &BuildError{
		Message:    "Dockerfile not found: " + path,
		Suggestion: "Create a Dockerfile or specify the correct path in .kudev.yaml",
	}
}

func ImageLoadFailed(cluster string, cause error) *BuildError {
	return &BuildError{
		Message:    "Failed to load image to " + cluster + " cluster",
		Suggestion: "Ensure your cluster is running and accessible",
		Cause:      cause,
	}
}

// Deploy errors

func DeploymentFailed(cause error) *DeployError {
	return &DeployError{
		Message:    "Failed to deploy to Kubernetes",
		Suggestion: "Check that your cluster is running and you have permissions",
		Cause:      cause,
	}
}

func DeploymentNotFound(name, namespace string) *DeployError {
	return &DeployError{
		Message:    "Deployment not found: " + namespace + "/" + name,
		Suggestion: "Run 'kudev up' to create the deployment first",
	}
}

func NamespaceCreateFailed(namespace string, cause error) *DeployError {
	return &DeployError{
		Message:    "Failed to create namespace: " + namespace,
		Suggestion: "Check that you have permissions to create namespaces",
		Cause:      cause,
	}
}

func PortForwardFailed(port int32, cause error) *DeployError {
	return &DeployError{
		Message:    fmt.Sprintf("Port forwarding failed on port %d", port),
		Suggestion: fmt.Sprintf("Port %d may be in use. Try a different port with --local-port", port),
		Cause:      cause,
	}
}

// Watch errors

func WatcherFailed(cause error) *WatchError {
	return &WatchError{
		Message:    "File watcher failed",
		Suggestion: "You may have too many files. Try adding exclusions to .kudev.yaml",
		Cause:      cause,
	}
}
