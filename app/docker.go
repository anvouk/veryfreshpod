package app

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Docker struct {
	client *client.Client
	logger *zap.SugaredLogger
}

func NewDockerClient(logger *zap.SugaredLogger) (*Docker, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to docker, is docker available?, %v", err)
	}

	docker := &Docker{
		client: dockerClient,
		logger: logger.Named("docker"),
	}
	return docker, nil
}

func (d *Docker) NegotiateVersion(ctx context.Context) {
	d.client.NegotiateAPIVersion(ctx)
}
