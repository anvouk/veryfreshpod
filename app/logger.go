package app

import (
	"go.uber.org/zap"
	"log"
)

func NewSugaredLogger(config *Config) *zap.SugaredLogger {
	var baseLogger *zap.Logger
	var err error
	if config.Debug {
		devConfig := zap.NewDevelopmentConfig()
		devConfig.Encoding = "console"
		baseLogger, err = devConfig.Build()
	} else {
		prodConfig := zap.NewProductionConfig()
		prodConfig.Encoding = "console"
		baseLogger, err = prodConfig.Build()
	}
	if err != nil {
		log.Fatalf("failed creating zap logger: %v", err)
	}
	return baseLogger.Sugar()
}
