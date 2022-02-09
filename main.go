package main

import (
	"github.com/anvouk/veryfreshpod/app"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"log"
)

func main() {
	// read env vars config
	var config app.Config
	if err := envconfig.Process("VFP", &config); err != nil {
		log.Fatalf("failed parsing env vars: %v", err)
	}

	// create zap logger (core)
	var baseLogger *zap.Logger
	var err error
	if config.Debug {
		baseLogger, err = zap.NewDevelopment()
	} else {
		baseLogger, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("failed creating zap logger: %v", err)
	}
	defer baseLogger.Sync()

	// cretate zap logger (pretty)
	logger := baseLogger.Sugar()

	logger.Infof("Debug %v!", config.Debug)
}
