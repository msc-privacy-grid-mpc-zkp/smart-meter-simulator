package main

import (
	"fmt"
	"time"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
)

func main() {
	fmt.Println("⚡ Starting Smart Meter Edge Simulator...")
	fmt.Println("------------------------------------------------")

	myMeter := meter.NewSimulatedMeter(500, 2000)

	for i := 1; i <= 5; i++ {
		reading := myMeter.Generate()

		fmt.Printf("[Reading %d] Timestamp: %d | Consumption: %d W\n",
			i, reading.Timestamp, reading.Consumption)

		time.Sleep(1 * time.Second)
	}

	fmt.Println("------------------------------------------------")
	fmt.Println("Simulation complete.")
}
