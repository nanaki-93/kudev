package config

func ApplyDefaults(cfg *DeploymentConfig) {
	if cfg == nil {
		return
	}

	if cfg.APIVersion == "" {
		cfg.APIVersion = DefaultAPIVersion
	}
	if cfg.Kind == "" {
		cfg.Kind = DefaultKind
	}
	if cfg.Spec.Namespace == "" {
		cfg.Spec.Namespace = "default"
	}

	if cfg.Spec.Replicas <= 0 {
		cfg.Spec.Replicas = 1
	}

	if cfg.Spec.LocalPort <= 0 {
		cfg.Spec.LocalPort = 8080
	}
	if cfg.Spec.ServicePort <= 0 {
		cfg.Spec.ServicePort = 8080
	}

	// Environment variables (empty is OK, no defaults)

	// KubeContext (empty is OK, uses whitelist validation)

	// BuildContextExclusions (empty is OK, just won't exclude extra files)
}
