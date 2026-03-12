package main

import (
	"fmt"
	"log"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/config"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/worker"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

func main() {
	fmt.Println("⚡ Starting Edge Simulator with Worker Pool architecture...")
	fmt.Println("---------------------------------------------------------")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Fatal error loading configuration: %v", err)
	}

	fmt.Println("[SETUP] Initializing ZKP Engine...")
	zkpEngine, err := zkp.Setup()
	if err != nil {
		log.Fatalf("Fatal ZKP setup error: %v", err)
	}

	clientA := network.NewCloudClient(cfg.Network.CloudURLA)
	clientB := network.NewCloudClient(cfg.Network.CloudURLB)

	var meters []*meter.SimulatedMeter
	for i := 1; i <= cfg.Simulation.MeterCount; i++ {
		meters = append(meters, meter.NewSimulatedMeter(cfg.Consumption.BaseLoad, cfg.Consumption.Variance))
	}

	queueSize := cfg.Simulation.MeterCount * 2
	pool := worker.NewPool(
		cfg.Simulation.WorkerPoolSize,
		queueSize,
		cfg.Consumption.MaxLimit,
		zkpEngine,
		clientA,
		clientB,
	)
	pool.Start()

	ticker := time.NewTicker(time.Duration(cfg.Simulation.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("\n--- New synchronized reading cycle ---")
		for i, m := range meters {
			pool.Jobs <- worker.Job{
				MeterID: fmt.Sprintf("meter-RS-%03d", i+1),
				Reading: m.Generate(),
			}
		}
	}
}
