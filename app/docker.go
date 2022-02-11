package app

import (
	"context"
	"fmt"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Docker struct {
	logger *zap.SugaredLogger
	client *client.Client
}

func NewDockerClient(logger *zap.SugaredLogger, ctx context.Context) (*Docker, error) {
	namedLogger := logger.Named("docker")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("failed connecting to docker, is docker available?, %v", err)
	}

	ver, err := dockerClient.ServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed detecting docker version: %v", err)
	}
	namedLogger.Infof("connected to docker: %v", ver.Version)

	dockerClient.NegotiateAPIVersion(ctx)

	docker := &Docker{
		logger: namedLogger,
		client: dockerClient,
	}

	return docker, nil
}
