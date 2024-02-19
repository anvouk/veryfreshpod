package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

type Docker struct {
	logger *zap.SugaredLogger
	client *client.Client
}

func NewDockerClient(logger *zap.SugaredLogger, ctx context.Context) (*Docker, error) {
	namedLogger := logger.Named("docker")
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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

func (docker *Docker) GetAllImages(ctx context.Context) ([]string, error) {
	images, err := docker.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, err
	}

	var imagesNames []string
	for _, image := range images {
		if len(image.RepoTags) != 0 {
			imagesNames = append(imagesNames, image.RepoTags[0])
		}
	}
	return imagesNames, nil
}

type DockerImagesWatcher struct {
	OnNewImage     func(imageName string, imageTag string)
	OnRemovedImage func(imageName string, imageTag string)
}

// ConvertImageNameToTag converts the pretty docker image name to sha tag.
// e.g. alpine:3.10 -> sha256:f8c20f8bbcb684055b4fea470fdd169c86e87786940b3262335b12ec3adef418
func (docker *Docker) ConvertImageNameToTag(ctx context.Context, imageName string) (string, error) {
	imageInfo, _, err := docker.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return "", err
	}
	return imageInfo.ID, nil
}

// ConvertImageTagToName converts the sha tag to pretty docker image name.
// e.g. sha256:f8c20f8bbcb684055b4fea470fdd169c86e87786940b3262335b12ec3adef418 -> alpine:3.10
func (docker *Docker) ConvertImageTagToName(ctx context.Context, imageTag string) (string, error) {
	imageInfo, _, err := docker.client.ImageInspectWithRaw(ctx, imageTag)
	if err != nil {
		return "", err
	}

	if len(imageInfo.RepoTags) == 0 {
		return "", nil
	}
	return imageInfo.RepoTags[0], nil
}

func (docker *Docker) WatchForImagesChanges(ctx context.Context, watcherInfo *DockerImagesWatcher) error {
	if watcherInfo == nil {
		return errors.New("watcherInfo cannot be nil")
	}

	go func() {
		msgs, errs := docker.client.Events(ctx, types.EventsOptions{})
	out:
		for {
			select {
			case err := <-errs:
				docker.logger.Infow("docker error event", "error", err)
				break out
			case msg := <-msgs:
				//docker.logger.Debugf("event: %+v", msg)
				// TODO: handle image load case
				switch msg.Action {
				case events.ActionPull:
					// {Status:pull ID:alpine:3.10 From: Type:image Action:pull Actor:{ID:alpine:3.10 Attributes:map[name:alpine]} Scope:local Time:1708196718 TimeNano:1708196718436679144}
					if watcherInfo.OnNewImage != nil {
						// we need to find the actual image tag
						imageName := msg.Actor.ID
						imageTag, err := docker.ConvertImageNameToTag(ctx, imageName)
						if err != nil {
							docker.logger.Errorw("failed to inspect new docker image", "error", err)
						} else {
							watcherInfo.OnNewImage(imageName, imageTag)
						}
					}
				case events.ActionTag:
					// {Status:tag ID:sha256:f8c20f8bbcb684055b4fea470fdd169c86e87786940b3262335b12ec3adef418 From: Type:image Action:tag Actor:{ID:sha256:f8c20f8bbcb684055b4fea470fdd169c86e87786940b3262335b12ec3adef418 Attributes:map[name:alpine-test:latest]} Scope:local Time:1708197072 TimeNano:1708197072385988042}
					if watcherInfo.OnNewImage != nil {
						watcherInfo.OnNewImage(msg.Actor.Attributes["name"], msg.Actor.ID)
					}
				}
			}
		}
	}()

	return nil
}
