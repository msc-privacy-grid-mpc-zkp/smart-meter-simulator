package main

import (
	"flag"
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

	// Parse CLI flags for red team testing
	maliciousReplay := flag.Bool("malicious-replay", false, "Enable Replay Attack simulation (Test 1.1)")
	maliciousTamperCount := flag.Int("malicious-tamper-count", 0, "Number of meters to tamper (Test 1.2, default 0 = disabled)")
	maliciousNoise := flag.Bool("malicious-noise", false, "Enable Random Noise (Invalid Proof) simulation (Test 1.3)")
	maliciousPoisoningCount := flag.Int("malicious-poisoning-count", 0, "Number of meters to poison with invalid shares (Test 2.1, default 0 = disabled)")
	maliciousOverflowCount := flag.Int("malicious-overflow-count", 0, "Number of overflow attack cycles (Test 2.2, default 0 = disabled)")
	flag.Parse()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("[FATAL] Error loading configuration: %v", err)
	}

	// Override config with CLI flags if provided
	cfg.RedTeam.MaliciousReplay = *maliciousReplay
	if cfg.RedTeam.MaliciousReplay {
		log.Println("[RED TEAM] ⚠️  REPLAY ATTACK SIMULATION ENABLED (Test 1.1)")
	}

	cfg.RedTeam.MaliciousTamperCount = *maliciousTamperCount
	if cfg.RedTeam.MaliciousTamperCount > 0 {
		log.Printf("[RED TEAM] ⚠️  PUBLIC INPUT TAMPERING SIMULATION ENABLED (Test 1.2) - Will tamper %d meter(s)\n", cfg.RedTeam.MaliciousTamperCount)
	}

	cfg.RedTeam.MaliciousNoise = *maliciousNoise
	if cfg.RedTeam.MaliciousNoise {
		log.Println("[RED TEAM] ⚠️  RANDOM NOISE (INVALID PROOF) SIMULATION ENABLED (Test 1.3)")
	}

	cfg.RedTeam.MaliciousPoisoningCount = *maliciousPoisoningCount
	if cfg.RedTeam.MaliciousPoisoningCount > 0 {
		log.Printf("[RED TEAM] ⚠️  DATA POISONING (INVALID SHARES) SIMULATION ENABLED (Test 2.1) - Will poison %d meter(s)\n", cfg.RedTeam.MaliciousPoisoningCount)
	}

	cfg.RedTeam.MaliciousOverflowCount = *maliciousOverflowCount
	if cfg.RedTeam.MaliciousOverflowCount > 0 {
		log.Printf("[RED TEAM] ⚠️  INTEGER OVERFLOW ATTEMPT SIMULATION ENABLED (Test 2.2) - Will trigger %d overflow cycle(s)\n", cfg.RedTeam.MaliciousOverflowCount)
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
		cfg.RedTeam.MaliciousReplay,
		cfg.RedTeam.MaliciousTamperCount,
		cfg.RedTeam.MaliciousNoise,
		cfg.RedTeam.MaliciousPoisoningCount,
		cfg.RedTeam.MaliciousOverflowCount,
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
			
			// RED TEAM: Reset tamperedCount at the beginning of each cycle
			// This ensures that exactly MaliciousTamperCount meters are tampered in EVERY cycle
			if cfg.RedTeam.MaliciousTamperCount > 0 {
				worker.ResetTamperedCount()
			}

			// RED TEAM: Reset poisonedCount at the beginning of each cycle
			// This ensures that exactly MaliciousPoisoningCount meters are poisoned in EVERY cycle
			if cfg.RedTeam.MaliciousPoisoningCount > 0 {
				worker.ResetPoisonedCount()
			}

			// RED TEAM: Reset overflowCount at the beginning of each cycle
			// This ensures that exactly MaliciousOverflowCount overflow cycles are triggered
			if cfg.RedTeam.MaliciousOverflowCount > 0 {
				worker.ResetOverflowCount()
			}
			
			for i, m := range meters {
				pool.Jobs <- worker.Job{
					MeterID: fmt.Sprintf("meter-RS-%03d", i+1),
					Reading: m.Generate(),
				}
			}

			// RED TEAM: Test 2.2 - Integer Overflow Attempt
			// If enabled, inject 10+ concurrent payloads with consumption at physical limit
			if cfg.RedTeam.MaliciousOverflowCount > 0 && worker.GetOverflowCount() < int32(cfg.RedTeam.MaliciousOverflowCount) {
				worker.IncrementOverflowCount()
				log.Printf("[RED TEAM] 🔴 Triggering Integer Overflow Attack cycle [%d/%d]\n", worker.GetOverflowCount(), cfg.RedTeam.MaliciousOverflowCount)
				
				// Generate 10+ payloads with consumption at the physical limit (MaxLimit)
				for j := 0; j < 10; j++ {
					pool.Jobs <- worker.Job{
						MeterID: fmt.Sprintf("meter-OVERFLOW-%03d", j+1),
						Reading: meter.Reading{
							Timestamp:   time.Now().Unix(),
							Consumption: cfg.Consumption.MaxLimit, // Physical limit (e.g., 10,000 W)
						},
					}
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
