package main

import (
	"context"
	"github.com/anvouk/veryfreshpod/app"
	appsV1 "k8s.io/api/apps/v1"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := app.NewConfig()
	logger := app.NewSugaredLogger(config)

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

	k8s, err := app.NewK8sClient(logger)
	if err != nil {
		logger.Fatalw("failed creating k8s client", "error", err)
	}

	if err := k8s.IsClusterVersionSupported(); err != nil {
		logger.Fatalw("unsupported k8s version or connection failure", "error", err)
	}

	_, err = app.NewDockerClient(logger, ctx)
	if err != nil {
		logger.Fatalw("failed creating docker client", "error", err)
	}

	if err := k8s.RegisterWatchForChanges(config, &app.K8sWatcher{
		OnAddDeployment: func(deployment *appsV1.Deployment) {
			logger.Infow("added deployment",
				"name", deployment.Name, "namespace", deployment.Namespace)
			containers := deployment.Spec.Template.Spec.Containers
			for _, container := range containers {
				logger.Infow("found images", "image", container.Image, "name", container.Name)
			}
		},
		OnRemoveDeployment: func(deployment *appsV1.Deployment) {
			logger.Infow("removed deployment",
				"name", deployment.Name, "namespace", deployment.Namespace)
		},
		OnAddStatefulSet: func(statefulSet *appsV1.StatefulSet) {
			logger.Infow("added deployment",
				"name", statefulSet.Name, "namespace", statefulSet.Namespace)
			containers := statefulSet.Spec.Template.Spec.Containers
			for _, container := range containers {
				logger.Infow("found images", "image", container.Image, "name", container.Name)
			}
		},
		OnRemoveStatefulSet: func(statefulSet *appsV1.StatefulSet) {
			logger.Infow("removed statefulSet",
				"name", statefulSet.Name, "namespace", statefulSet.Namespace)
		},
	}); err != nil {
		logger.Fatalw("failed registering callbacks for k8s watcher", "error", err)
	}

	if err := k8s.Run(ctx); err != nil {
		logger.Fatalw("failed listening for k8s changes", "error", err)
	}
	<-ctx.Done()
}
