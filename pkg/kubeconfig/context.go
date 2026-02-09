package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/tools/clientcmd"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type Context struct {
	Name          string
	ClusterName   string
	ClusterServer string
	Username      string
}

func LoadCurrentContext() (*Context, error) {
	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	currentContext := config.CurrentContext
	if currentContext == "" {
		return nil, fmt.Errorf("no current context found in kubeconfig (%s)\n\n"+
			"Set current context with: kubectl config use-context <context-name>\n"+
			"Available context: %v", kubeconfigPath, getAvailableContextNames(config))
	}

	context, ok := config.Contexts[currentContext]
	if !ok {
		return nil, fmt.Errorf("current context %q not found in kubeconfig (%s)", currentContext, kubeconfigPath)
	}
	clustername := context.Cluster
	clusterServer := ""

	if cluster, ok := config.Clusters[clustername]; ok {
		clusterServer = cluster.Server
	}

	return &Context{
			Name:          currentContext,
			ClusterName:   clustername,
			ClusterServer: clusterServer,
			Username:      ""},
		nil
}

func ListAvailableContexts() ([]string, error) {
	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}
	return getAvailableContextNames(config), nil
}

func ContextExists(contextName string) (bool, error) {

	kubeconfigPath, err := getKubeconfigPath()
	if err != nil {
		return false, err
	}
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return false, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}
	_, exists := config.Contexts[contextName]
	return exists, nil
}

func getKubeconfigPath() (string, error) {
	if kubeConfig := os.Getenv("KUBECONFIG"); kubeConfig != "" {
		return kubeConfig, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	defaultKubeConfigPath := filepath.Join(homeDir, ".kube", "config")
	if _, err := os.Stat(defaultKubeConfigPath); err == nil {
		return defaultKubeConfigPath, nil
	}
	return "", fmt.Errorf("kubeconfig not found\n\n"+
		"Kubeconfig locations checked:\n"+
		" - $KUBECONFIG environment variable "+
		" - %s (default)\n\n"+
		"Setup: mkdir -p ~/.kube && kubectl config view > ~/.kube/config",
		defaultKubeConfigPath)
}

func getAvailableContextNames(config *clientcmdapi.Config) []string {
	names := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		names = append(names, name)
	}
	return names
}
func GetKubeconfigPath() (string, error) {
	return getKubeconfigPath()
}
