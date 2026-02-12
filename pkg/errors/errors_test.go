package errors

import (
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	err := ConfigNotFound("/path/to/.kudev.yaml")

	if err.ExitCode() != ExitConfig {
		t.Errorf("ExitCode() = %d, want %d", err.ExitCode(), ExitConfig)
	}

	if err.UserMessage() == "" {
		t.Error("UserMessage() should not be empty")
	}

	if err.SuggestedAction() == "" {
		t.Error("SuggestedAction() should not be empty")
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	err := DockerBuildFailed(cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestKudevErrorInterface(t *testing.T) {
	tests := []struct {
		name     string
		err      KudevError
		exitCode int
	}{
		{"ConfigError", ConfigNotFound("x"), ExitConfig},
		{"KubeAuthError", KubeconfigNotFound(), ExitKubeAuth},
		{"BuildError", DockerNotRunning(nil), ExitBuild},
		{"DeployError", DeploymentNotFound("x", "y"), ExitDeploy},
		{"WatchError", WatcherFailed(nil), ExitWatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ExitCode() != tt.exitCode {
				t.Errorf("ExitCode() = %d, want %d", tt.err.ExitCode(), tt.exitCode)
			}

			if tt.err.UserMessage() == "" {
				t.Error("UserMessage() should not be empty")
			}
		})
	}
}
