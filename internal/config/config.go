package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	CloudURL        string `yaml:"cloud_url"`
	MeterCount      int    `yaml:"meter_count"`
	WorkerPoolSize  int    `yaml:"worker_pool_size"`
	IntervalSeconds int    `yaml:"interval_seconds"`
	BaseLoad        uint64 `yaml:"base_load"`
	Variance        uint64 `yaml:"variance"`
	MaxLimit        uint64 `yaml:"max_limit"`
}

func LoadConfig(path string) (*AppConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			log.Printf("warning: failed to close config file: %v\n", closeErr)
		}
	}()

	var cfg AppConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.WorkerPoolSize < 1 {
		cfg.WorkerPoolSize = 1
	}
	if cfg.MeterCount < 1 {
		cfg.MeterCount = 1
	}

	return &cfg, nil
}
