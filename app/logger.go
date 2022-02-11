package app

import (
	"go.uber.org/zap"
	"log"
)

func getLoggerEncoding(config *Config) string {
	if config.LoggerUseJson {
		return "json"
	}
	return "console"
}

func NewSugaredLogger(config *Config) *zap.SugaredLogger {
	var baseLogger *zap.Logger
	var err error
	if config.Debug {
		devConfig := zap.NewDevelopmentConfig()
		devConfig.Encoding = getLoggerEncoding(config)
		baseLogger, err = devConfig.Build()
	} else {
		prodConfig := zap.NewProductionConfig()
		prodConfig.Encoding = getLoggerEncoding(config)
		baseLogger, err = prodConfig.Build()
	}
	if err != nil {
		log.Fatalf("failed creating zap logger: %v", err)
	}
	return baseLogger.Sugar()
}
