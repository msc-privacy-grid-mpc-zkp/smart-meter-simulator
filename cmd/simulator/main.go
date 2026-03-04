package main

import (
	"fmt"
	"log"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

func main() {
	fmt.Println("⚡ Starting Smart Meter Edge Simulator...")
	fmt.Println("------------------------------------------------")

	fmt.Println("[1/3] Initializing ZKP Engine (Trusted Setup)...")
	setupStartTime := time.Now()
	zkpEngine, err := zkp.Setup()
	if err != nil {
		log.Fatalf("Fatal error during ZKP setup: %v", err)
	}
	fmt.Printf("      [OK] Setup completed in %v\n", time.Since(setupStartTime))

	fmt.Println("[2/3] Initializing Smart Meter Sensor...")
	myMeter := meter.NewSimulatedMeter(500, 2000)

	const maxLimit uint64 = 10000

	fmt.Println("[3/3] Starting Simulation Loop...")
	fmt.Println("------------------------------------------------")

	for i := 1; i <= 5; i++ {
		reading := myMeter.Generate()
		fmt.Printf("[Reading %d] Time: %d | Consumption: %d W\n",
			i, reading.Timestamp, reading.Consumption)

		proofStartTime := time.Now()
		_, err := zkpEngine.GenerateProof(reading.Consumption, maxLimit)
		if err != nil {
			log.Printf("      [ERROR] Failed to generate ZKP proof: %v\n", err)
			continue
		}

		fmt.Printf("      [SUCCESS] ZKP Proof generated in %v!\n", time.Since(proofStartTime))

		time.Sleep(1 * time.Second)
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("Simulation complete. All proofs generated successfully.")
}
