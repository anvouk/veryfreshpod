package app

import (
	"go.uber.org/zap"
	"log"
)

func NewSugaredLogger(config *Config) *zap.SugaredLogger {
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
	return baseLogger.Sugar()
}
