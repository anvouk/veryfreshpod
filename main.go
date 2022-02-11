package main

import (
	"context"
	"github.com/anvouk/veryfreshpod/app"
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

	controller := k8s.WatchPodsForChanges(config)
	controller.Run(ctx.Done())
}
