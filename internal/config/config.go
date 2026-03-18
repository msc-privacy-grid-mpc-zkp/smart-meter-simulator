package config

import (
	"flag"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

// AppConfig maps YAML sections and Environment variables
type AppConfig struct {
	Network struct {
		CloudURLs []string `yaml:"cloud_urls" env:"CLOUD_URLS" env-delim:","`
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

// LoadConfig reads configuration from file or environment variables
func LoadConfig() (*AppConfig, error) {
	configPath := flag.String("config", "config.yaml", "Path to YAML configuration")
	flag.Parse()

	var cfg AppConfig

	err := cleanenv.ReadConfig(*configPath, &cfg)
	if err != nil {
		log.Println("[INFO] YAML config not found, falling back to Environment variables.")
		if errEnv := cleanenv.ReadEnv(&cfg); errEnv != nil {
			return nil, errEnv
		}
	}

	// Safety checks (fallback to defaults if completely missing)
	if cfg.Simulation.WorkerPoolSize < 1 {
		cfg.Simulation.WorkerPoolSize = 1
	}
	if cfg.Simulation.MeterCount < 1 {
		cfg.Simulation.MeterCount = 1
	}

	return &cfg, nil
}
