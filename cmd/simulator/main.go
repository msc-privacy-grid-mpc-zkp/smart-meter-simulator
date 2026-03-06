package main

import (
	"fmt"
	"log"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

func main() {
	fmt.Println("⚡ Starting Smart Meter Edge Simulator...")
	fmt.Println("------------------------------------------------")

	fmt.Println("[1/4] Initializing ZKP Engine (Trusted Setup)...")
	setupStartTime := time.Now()
	zkpEngine, err := zkp.Setup()
	if err != nil {
		log.Fatalf("Fatal error during ZKP setup: %v", err)
	}
	fmt.Printf("      [OK] Setup completed in %v\n", time.Since(setupStartTime))

	fmt.Println("[2/4] Initializing Smart Meter Sensor...")
	myMeter := meter.NewSimulatedMeter(500, 2000)
	meterID := "meter-RS-001"
	const maxLimit uint64 = 10000

	fmt.Println("[3/4] Initializing Network Client...")

	cloudURL := "http://localhost:8080/api/proofs"
	apiClient := network.NewCloudClient(cloudURL)
	fmt.Printf("      [OK] Client configured for %s\n", cloudURL)

	fmt.Println("[4/4] Starting Simulation Loop...")
	fmt.Println("------------------------------------------------")

	for i := 1; i <= 5; i++ {
		fmt.Printf("\n--- Cycle %d ---\n", i)

		reading := myMeter.Generate()
		fmt.Printf("[SENSOR] Time: %d | Consumption: %d W\n", reading.Timestamp, reading.Consumption)

		proofStartTime := time.Now()
		proof, err := zkpEngine.GenerateProof(reading.Consumption, maxLimit)
		if err != nil {
			log.Printf("[ERROR] Failed to generate ZKP proof: %v\n", err)
			continue
		}
		fmt.Printf("[ZKP]    Proof generated successfully in %v\n", time.Since(proofStartTime))

		proofBytes, err := network.SerializeProof(proof)
		if err != nil {
			log.Printf("[ERROR] Failed to serialize proof: %v\n", err)
			continue
		}

		payload := network.ProofPayload{
			MeterID:   meterID,
			Timestamp: reading.Timestamp,
			Proof:     proofBytes,
		}

		fmt.Printf("[NETWORK] Sending %d bytes to Cloud...\n", len(proofBytes))
		err = apiClient.SendProof(payload)
		if err != nil {
			log.Printf("[WARNING] Network error (Expected if Cloud is down): %v\n", err)
		} else {
			fmt.Println("[SUCCESS] Proof successfully accepted by the MPC Cloud!")
		}

		time.Sleep(2 * time.Second)
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("Simulation complete. Shutting down Edge node.")
}
