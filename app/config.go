package app

import (
	"github.com/kelseyhightower/envconfig"
	"log"
	"time"
)

type Config struct {
	Debug           bool          `split_words:"true" default:"true"`
	LoggerUseJson   bool          `split_words:"true" default:"true"`
	RefreshInterval time.Duration `split_words:"true" default:"5s"`
}

func NewConfig() *Config {
	var config Config
	if err := envconfig.Process("VFP", &config); err != nil {
		log.Fatalf("failed parsing env vars: %v", err)
	}
	return &config
}
