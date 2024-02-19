package main

import (
	"context"
	"github.com/anvouk/veryfreshpod/app"
	"go.uber.org/zap"
	appsV1 "k8s.io/api/apps/v1"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := app.NewConfig()
	logger := app.NewSugaredLogger(config)
	defer func(logger *zap.SugaredLogger) {
		_ = logger.Sync()
	}(logger)

	logger.Infof("veryfreshpod starting")
	logger.Infof("debug mode: %v", config.Debug)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan
		logger.Infof("received signal: %v", sig.String())
		cancel()
	}()

	docker, err := app.NewDockerClient(logger, ctx)
	if err != nil {
		logger.Fatalw("failed creating docker client", "error", err)
	}

	k8s, err := app.NewK8sClient(logger)
	if err != nil {
		logger.Fatalw("failed creating k8s client", "error", err)
	}

	if err := k8s.IsClusterVersionSupported(); err != nil {
		logger.Fatalw("unsupported k8s version or connection failure", "error", err)
	}

	store := app.NewStore(logger, docker, k8s)

	err = docker.WatchForImagesChanges(ctx, &app.DockerImagesWatcher{
		OnNewImage: func(imageName string, imageTag string) {
			logger.Debugw("new docker image", "imageName", imageName, "imageTag", imageTag)
			store.NewDockerImage(ctx, imageName)
		},
		OnRemovedImage: func(imageName string, imageTag string) {
			logger.Debugw("removed docker image", "imageName", imageName, "imageTag", imageTag)
			store.RemoveDockerImage(imageName)
		},
	})
	if err != nil {
		logger.Fatalw("failed watching docker for images changes", "error", err)
	}

	if err := k8s.RegisterWatchForChanges(config, &app.K8sWatcher{
		OnAddDeployment: func(deployment *appsV1.Deployment) {
			logger.Debugw("added deployment", "name", deployment.Name, "namespace", deployment.Namespace)
			store.NewK8sDeployment(deployment)
		},
		OnRemoveDeployment: func(deployment *appsV1.Deployment) {
			logger.Debugw("removed deployment", "name", deployment.Name, "namespace", deployment.Namespace)
			store.RemoveK8sDeployment(deployment)
		},
		OnAddStatefulSet: func(statefulSet *appsV1.StatefulSet) {
			logger.Debugw("added statefulSet", "name", statefulSet.Name, "namespace", statefulSet.Namespace)
			store.NewK8sStatefulSet(statefulSet)
		},
		OnRemoveStatefulSet: func(statefulSet *appsV1.StatefulSet) {
			logger.Debugw("removed statefulSet", "name", statefulSet.Name, "namespace", statefulSet.Namespace)
			store.RemoveK8sStatefulSet(statefulSet)
		},
	}); err != nil {
		logger.Fatalw("failed registering callbacks for k8s watcher", "error", err)
	}

	if err := k8s.Run(ctx); err != nil {
		logger.Fatalw("failed listening for k8s changes", "error", err)
	}
	<-ctx.Done()
}
