package config

type DeploymentConfig struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   MetadataConfig `yaml:"metadata"`
	Spec       SpecConfig     `yaml:"spec"`
}

type MetadataConfig struct {
	Name string `yaml:"name"`
}

type SpecConfig struct {
	ImageName              string   `yaml:"imageName"`
	DockerFilePath         string   `yaml:"dockerFilePath"`
	Namespace              string   `yaml:"namespace"`
	Replicas               int32    `yaml:"replicas"`
	LocalPort              int32    `yaml:"localPort"`
	ServicePort            int32    `yaml:"servicePort"`
	Env                    []EnvVar `yaml:"env"`
	KubeContext            string   `yaml:"kubeContext,omitempty"`
	BuildContextExclusions []string `yaml:"buildContextExclusions,omitempty"`
}

type EnvVar struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}
