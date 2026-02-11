package templates

import _ "embed"

//go:embed deployment.yaml
var DeploymentTemplate string

// ServiceTemplate is the embedded Service YAML template.
//
//go:embed service.yaml
var ServiceTemplate string
