package deployer

import (
	"bytes"
	"fmt"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

// Renderer handles YAML template rendering.
type Renderer struct {
	deploymentTpl *template.Template
	serviceTpl    *template.Template
}

// templateFuncs provides custom functions for templates.
var templateFuncs = template.FuncMap{
	// quote wraps a string in double quotes
	"quote": func(s string) string {
		return fmt.Sprintf("%q", s)
	},
	// default returns the default value if the input is empty
	"default": func(defaultVal, val interface{}) interface{} {
		if val == nil || val == "" {
			return defaultVal
		}
		return val
	},
}

// NewRenderer creates a new template renderer.
// deploymentTpl and serviceTpl are the raw template strings (from go:embed).
func NewRenderer(deploymentTpl, serviceTpl string) (*Renderer, error) {
	depTpl, err := template.New("deployment").
		Funcs(templateFuncs).
		Parse(deploymentTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment template: %w", err)
	}

	svcTpl, err := template.New("service").
		Funcs(templateFuncs).
		Parse(serviceTpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service template: %w", err)
	}

	return &Renderer{
		deploymentTpl: depTpl,
		serviceTpl:    svcTpl,
	}, nil
}

// RenderDeployment renders the Deployment template with the given data.
// Returns a typed Kubernetes Deployment object.
func (r *Renderer) RenderDeployment(data TemplateData) (*appsv1.Deployment, error) {
	// Validate input
	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid template data: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := r.deploymentTpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute deployment template: %w", err)
	}

	// Parse YAML into K8s object
	deployment := &appsv1.Deployment{}
	if err := yaml.Unmarshal(buf.Bytes(), deployment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment YAML: %w\nRendered YAML:\n%s",
			err, buf.String())
	}

	return deployment, nil
}

// RenderService renders the Service template with the given data.
// Returns a typed Kubernetes Service object.
func (r *Renderer) RenderService(data TemplateData) (*corev1.Service, error) {
	// Validate input
	if err := data.Validate(); err != nil {
		return nil, fmt.Errorf("invalid template data: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := r.serviceTpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute service template: %w", err)
	}

	// Parse YAML into K8s object
	service := &corev1.Service{}
	if err := yaml.Unmarshal(buf.Bytes(), service); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service YAML: %w\nRendered YAML:\n%s",
			err, buf.String())
	}

	return service, nil
}

// RenderDeploymentYAML renders the Deployment template and returns raw YAML.
// Useful for --dry-run output.
func (r *Renderer) RenderDeploymentYAML(data TemplateData) (string, error) {
	if err := data.Validate(); err != nil {
		return "", fmt.Errorf("invalid template data: %w", err)
	}

	var buf bytes.Buffer
	if err := r.deploymentTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute deployment template: %w", err)
	}

	return buf.String(), nil
}

// RenderServiceYAML renders the Service template and returns raw YAML.
// Useful for --dry-run output.
func (r *Renderer) RenderServiceYAML(data TemplateData) (string, error) {
	if err := data.Validate(); err != nil {
		return "", fmt.Errorf("invalid template data: %w", err)
	}

	var buf bytes.Buffer
	if err := r.serviceTpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute service template: %w", err)
	}

	return buf.String(), nil
}

// RenderAll renders both Deployment and Service, returning raw YAML.
// Useful for --dry-run to show complete manifests.
func (r *Renderer) RenderAll(data TemplateData) (string, error) {
	depYAML, err := r.RenderDeploymentYAML(data)
	if err != nil {
		return "", err
	}

	svcYAML, err := r.RenderServiceYAML(data)
	if err != nil {
		return "", err
	}

	// Combine with YAML document separator
	return fmt.Sprintf("%s---\n%s", depYAML, svcYAML), nil
}
