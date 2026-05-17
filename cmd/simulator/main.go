package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/config"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/worker"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

const (
	// BufferMultiplier ensures the job channel has enough capacity to hold
	// multiple cycles of readings, preventing the simulation loop from blocking
	// during slow network responses.
	BufferMultiplier = 2
)

func main() {
	fmt.Println("⚡ Starting Edge Simulator (Multi-Node MPC ready)...")
	fmt.Println("---------------------------------------------------------")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Error loading configuration: %v", err)
	}

	log.Println("[SETUP] Initializing ZKP Engine...")
	zkpEngine, err := zkp.Setup()
	if err != nil {
		log.Fatalf("[FATAL] ZKP setup error: %v", err)
	}

	var clients []*network.Client
	for _, url := range cfg.Network.AggregatorURLs {
		clients = append(clients, network.NewClient(url))
	}

	if len(clients) == 0 {
		log.Fatalf("[FATAL] No aggregator URLs found in configuration. Check your config.yaml or ENV variables.")
	}
	log.Printf("[NETWORK] Initialized %d MPC aggregator clients\n", len(clients))

	var meters []*meter.SimulatedMeter
	for i := 1; i <= cfg.Simulation.MeterCount; i++ {
		meters = append(meters, meter.NewSimulatedMeter(cfg.Consumption.BaseLoad, cfg.Consumption.Variance))
	}

	queueSize := cfg.Simulation.MeterCount * BufferMultiplier
	pool := worker.NewPool(
		cfg.Simulation.WorkerPoolSize,
		queueSize,
		cfg.Consumption.MaxLimit,
		zkpEngine,
		clients,
	)
	pool.Start()

	ticker := time.NewTicker(time.Duration(cfg.Simulation.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("[SYSTEM] Simulation running. Press Ctrl+C to stop.")

	for {
		select {
		case <-ticker.C:
			fmt.Println("\n--- New synchronized reading cycle ---")
			for i, m := range meters {
				pool.Jobs <- worker.Job{
					MeterID: fmt.Sprintf("meter-RS-%03d", i+1),
					Reading: m.Generate(),
				}
			}
		case sig := <-sigChan:
			log.Printf("\n[SYSTEM] Received OS signal: %v. Initiating graceful shutdown...\n", sig)
			
			ticker.Stop()

			close(pool.Jobs)

			log.Println("[SYSTEM] Waiting for workers to finish current tasks...")

			pool.Wait()

			log.Println("[SYSTEM] Edge Simulator stopped cleanly.")
			return
		}
	}
}
