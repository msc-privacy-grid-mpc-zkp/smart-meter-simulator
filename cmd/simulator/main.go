package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/config"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

// Job represents a task for the ZKP worker (a single reading to be proven and sent)
type Job struct {
	MeterID string
	Reading meter.Reading
}

// TODO: Implement gRPC and refactor main.go
func main() {
	fmt.Println("⚡ Starting Edge Simulator with Worker Pool architecture...")
	fmt.Println("---------------------------------------------------------")

	// 1. Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Fatal error loading configuration: %v", err)
	}
	fmt.Printf("[CONFIG] Loaded: %d meters | %d workers | Interval: %ds\n",
		cfg.MeterCount, cfg.WorkerPoolSize, cfg.IntervalSeconds)

	// 2. ZKP Setup (Done strictly once for the entire network - saves RAM and CPU)
	fmt.Println("[SETUP] Initializing ZKP Engine...")
	zkpEngine, err := zkp.Setup()
	if err != nil {
		log.Fatalf("Fatal ZKP setup error: %v", err)
	}

	// 3. Initialize network client
	apiClient := network.NewCloudClient(cfg.CloudURL)

	// 4. Create an array of our virtual meters
	var meters []*meter.SimulatedMeter
	for i := 1; i <= cfg.MeterCount; i++ {
		meters = append(meters, meter.NewSimulatedMeter(cfg.BaseLoad, cfg.Variance))
	}

	// 5. Create a channel for tasks (Queue)
	// It is buffered so the main loop doesn't block while generating jobs
	jobs := make(chan Job, cfg.MeterCount*2)

	// 6. Start Workers
	var wg sync.WaitGroup
	for w := 1; w <= cfg.WorkerPoolSize; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Worker continuously listens to the channel and waits for new jobs
			for job := range jobs {
				// A. Generate proof
				proof, err := zkpEngine.GenerateProof(job.Reading.Consumption, cfg.MaxLimit)
				if err != nil {
					log.Printf("[Worker %d] Error generating proof for %s: %v\n", workerID, job.MeterID, err)
					continue
				}

				// B. Serialize and package
				proofBytes, _ := network.SerializeProof(proof)
				payload := network.ProofPayload{
					MeterID:   job.MeterID,
					Timestamp: job.Reading.Timestamp,
					Proof:     proofBytes,
				}

				// C. Send to Cloud
				err = apiClient.SendProof(payload)
				if err != nil {
					// We expect an error until we set up the Cloud/MPC server
					log.Printf("[Worker %d] Network error (Expected) for %s: %v\n", workerID, job.MeterID, err)
				} else {
					fmt.Printf("[Worker %d] ZKP successfully generated and sent for %s!\n", workerID, job.MeterID)
				}
			}
		}(w)
	}

	// 7. Main simulation loop (Ticks every X seconds)
	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	fmt.Println("---------------------------------------------------------")
	fmt.Println("Simulation is active. Press Ctrl+C to stop.")

	for range ticker.C {
		fmt.Println("\n--- New synchronized reading cycle ---")

		// Iterate through each virtual meter
		for i, m := range meters {
			meterID := fmt.Sprintf("meter-RS-%03d", i+1)
			reading := m.Generate()

			// Send job to the channel (continue immediately, without waiting for the math)
			jobs <- Job{
				MeterID: meterID,
				Reading: reading,
			}
		}
	}
}
