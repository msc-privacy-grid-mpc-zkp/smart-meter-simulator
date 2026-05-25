package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/config"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/simulator"
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
	maliciousOverflowMeters := flag.Int("malicious-overflow-meters", 10, "Number of meters to send at MaxLimit per overflow cycle (default 10)")
	maliciousMixedTraffic := flag.Bool("malicious-mixed-traffic", false, "Enable Mixed Traffic (honest + overflow intermixed) simulation (Test 2.3)")
	maliciousMixedHonest := flag.Int("malicious-mixed-honest", 5, "Number of honest meters in mixed traffic batch (default 5)")
	maliciousMixedOverflow := flag.Int("malicious-mixed-overflow", 5, "Number of overflow meters in mixed traffic batch (default 5)")
	slowlorisEnabled := flag.Bool("slowloris", false, "Enable Slowloris DoS attack simulation (Test 3.1)")
	slowlorisConnections := flag.Int("slowloris-connections", 10, "Number of slow connections to maintain (default 10)")
	slowlorisDelaySeconds := flag.Int("slowloris-delay-seconds", 2, "Delay in seconds between sending bytes (default 2)")
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

	cfg.RedTeam.MaliciousOverflowMeters = *maliciousOverflowMeters
	if cfg.RedTeam.MaliciousOverflowMeters > 0 && cfg.RedTeam.MaliciousOverflowCount > 0 {
		log.Printf("[RED TEAM] Overflow meters per cycle: %d\n", cfg.RedTeam.MaliciousOverflowMeters)
	}

	cfg.RedTeam.MaliciousMixedTraffic = *maliciousMixedTraffic
	if cfg.RedTeam.MaliciousMixedTraffic {
		log.Printf("[RED TEAM] ⚠️  MIXED TRAFFIC (HONEST + OVERFLOW INTERMIXED) SIMULATION ENABLED (Test 2.3) - %d honest + %d overflow\n",
			*maliciousMixedHonest, *maliciousMixedOverflow)
	}

	cfg.RedTeam.MaliciousMixedHonest = *maliciousMixedHonest
	cfg.RedTeam.MaliciousMixedOverflow = *maliciousMixedOverflow

	cfg.RedTeam.SlowlorisEnabled = *slowlorisEnabled
	if cfg.RedTeam.SlowlorisEnabled {
		log.Printf("[RED TEAM] ⚠️  SLOWLORIS DoS ATTACK SIMULATION ENABLED (Test 3.1) - %d connections with %d second delay\n",
			*slowlorisConnections, *slowlorisDelaySeconds)
	}

	cfg.RedTeam.SlowlorisConnections = *slowlorisConnections
	cfg.RedTeam.SlowlorisDelaySeconds = *slowlorisDelaySeconds

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
		cfg.RedTeam.MaliciousOverflowMeters,
	)
	pool.Start()

	ticker := time.NewTicker(time.Duration(cfg.Simulation.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("[SYSTEM] Simulation running. Press Ctrl+C to stop.")

	// RED TEAM: Test 3.1 - Slowloris DoS Attack
	// If enabled, start the slowloris attack in a separate goroutine
	if cfg.RedTeam.SlowlorisEnabled {
		go func() {
			time.Sleep(2 * time.Second) // Wait for aggregator to be ready
			slowlorisAttack := simulator.NewSlowlorisAttack(
				cfg.Network.AggregatorURLs[0],
				cfg.RedTeam.SlowlorisConnections,
				cfg.RedTeam.SlowlorisDelaySeconds,
			)
			slowlorisAttack.Start()
		}()
	}

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

			// RED TEAM: Test 2.3 - Mixed Traffic (Honest + Overflow Intermixed)
			// If enabled, create a unified batch of honest and overflow meters with identical timestamps
			if cfg.RedTeam.MaliciousMixedTraffic {
				log.Printf("[RED TEAM] 🔴 Triggering Mixed Traffic Attack: %d honest + %d overflow meters (intermixed)\n",
					cfg.RedTeam.MaliciousMixedHonest, cfg.RedTeam.MaliciousMixedOverflow)

				// Create a unified timestamp for all payloads in this batch
				unifiedTimestamp := time.Now().Unix()

				// Create a list of jobs: honest + overflow
				var mixedJobs []worker.Job

				// Add honest meters (normal consumption)
				for i := 0; i < cfg.RedTeam.MaliciousMixedHonest; i++ {
					mixedJobs = append(mixedJobs, worker.Job{
						MeterID: fmt.Sprintf("meter-MIXED-HONEST-%03d", i+1),
						Reading: meter.Reading{
							Timestamp:   unifiedTimestamp,
							Consumption: cfg.Consumption.BaseLoad + (uint64(i) % cfg.Consumption.Variance),
						},
					})
				}

				// Add overflow meters (at MaxLimit)
				for j := 0; j < cfg.RedTeam.MaliciousMixedOverflow; j++ {
					mixedJobs = append(mixedJobs, worker.Job{
						MeterID: fmt.Sprintf("meter-MIXED-OVERFLOW-%03d", j+1),
						Reading: meter.Reading{
							Timestamp:   unifiedTimestamp,
							Consumption: cfg.Consumption.MaxLimit,
						},
					})
				}

				// Shuffle the jobs to intermix honest and overflow
				rand.Shuffle(len(mixedJobs), func(i, j int) {
					mixedJobs[i], mixedJobs[j] = mixedJobs[j], mixedJobs[i]
				})

				// Dispatch all shuffled jobs
				for _, job := range mixedJobs {
					pool.Jobs <- job
				}
			} else {
				// Normal operation: dispatch regular meters
				for i, m := range meters {
					pool.Jobs <- worker.Job{
						MeterID: fmt.Sprintf("meter-RS-%03d", i+1),
						Reading: m.Generate(),
					}
				}

				// RED TEAM: Test 2.2 - Integer Overflow Attempt
				// If enabled, inject N concurrent payloads with consumption at physical limit
				if cfg.RedTeam.MaliciousOverflowCount > 0 && worker.GetOverflowCount() < int32(cfg.RedTeam.MaliciousOverflowCount) {
					worker.IncrementOverflowCount()
					log.Printf("[RED TEAM] 🔴 Triggering Integer Overflow Attack cycle [%d/%d] with %d meters\n",
						worker.GetOverflowCount(), cfg.RedTeam.MaliciousOverflowCount, cfg.RedTeam.MaliciousOverflowMeters)

					// Generate N payloads with consumption at the physical limit (MaxLimit)
					for j := 0; j < cfg.RedTeam.MaliciousOverflowMeters; j++ {
						pool.Jobs <- worker.Job{
							MeterID: fmt.Sprintf("meter-OVERFLOW-%03d", j+1),
							Reading: meter.Reading{
								Timestamp:   time.Now().Unix(),
								Consumption: cfg.Consumption.MaxLimit, // Physical limit (e.g., 10,000 W)
							},
						}
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
