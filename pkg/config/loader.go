package config

import "context"

type LoaderConfig interface {
	Load(ctx context.Context) (*DeploymentConfig, error)
	Save(ctx context.Context, config *DeploymentConfig) error
}
