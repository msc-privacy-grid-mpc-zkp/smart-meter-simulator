package config

import (
	"flag"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

const defaultConfigFile = "config.yaml"

// AppConfig represents the central configuration structure for the simulator,
// mapping both YAML file sections and OS Environment variables.
type AppConfig struct {
	Network struct {
		AggregatorURLs []string `yaml:"aggregator_urls" env:"AGGREGATOR_URLS" env-delim:","`
	} `yaml:"network"`

	Simulation struct {
		MeterCount      int `yaml:"meter_count" env:"METER_COUNT" env-default:"10"`
		WorkerPoolSize  int `yaml:"worker_pool_size" env:"WORKER_POOL_SIZE" env-default:"4"`
		IntervalSeconds int `yaml:"interval_seconds" env:"INTERVAL_SECONDS" env-default:"5"`
	} `yaml:"simulation"`

	Consumption struct {
		BaseLoad uint64 `yaml:"base_load" env:"BASE_LOAD" env-default:"500"`
		Variance uint64 `yaml:"variance" env:"VARIANCE" env-default:"2000"`
		MaxLimit uint64 `yaml:"max_limit" env:"MAX_LIMIT" env-default:"10000"`
	} `yaml:"consumption"`
}

// LoadConfig parses the configuration from the specified YAML file or
// falls back to environment variables if the file is missing.
// It enforces minimum safety bounds for critical simulation parameters.
func LoadConfig() (*AppConfig, error) {
	configPath := flag.String("config", defaultConfigFile, "Path to the YAML configuration file")
	flag.Parse()

	var cfg AppConfig

	if err := cleanenv.ReadConfig(*configPath, &cfg); err != nil {
		log.Printf("[INFO] YAML config (%s) not found or invalid, falling back to Environment variables.\n", *configPath)
		if errEnv := cleanenv.ReadEnv(&cfg); errEnv != nil {
			return nil, errEnv
		}
	}

	if cfg.Simulation.WorkerPoolSize < 1 {
		cfg.Simulation.WorkerPoolSize = 1
	}
	if cfg.Simulation.MeterCount < 1 {
		cfg.Simulation.MeterCount = 1
	}
	if cfg.Simulation.IntervalSeconds < 1 {
		cfg.Simulation.IntervalSeconds = 1
	}

	return &cfg, nil
}
