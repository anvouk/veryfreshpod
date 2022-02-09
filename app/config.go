package app

import (
	"github.com/kelseyhightower/envconfig"
	"log"
)

type Config struct {
	Debug bool `split_words:"true" default:"true"`
}

func NewConfig() *Config {
	var config Config
	if err := envconfig.Process("VFP", &config); err != nil {
		log.Fatalf("failed parsing env vars: %v", err)
	}
	return &config
}
