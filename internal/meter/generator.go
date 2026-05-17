package meter

import (
	"math/rand"
	"time"
)

// Reading represents a single data point recorded by a smart meter.
type Reading struct {
	Timestamp   int64  // Unix epoch time when the reading was taken
	Consumption uint64 // Power consumption value (e.g., in Watt-hours)
}

// Generator defines the contract for any smart meter implementation
// capable of producing sequential consumption readings.
type Generator interface {
	Generate() Reading
}

// SimulatedMeter implements the Generator interface to produce synthetic
// power consumption data for load testing and simulation purposes.
type SimulatedMeter struct {
	BaseLoad uint64
	Variance uint64
}

// NewSimulatedMeter initializes a new synthetic meter with a specific load profile.
func NewSimulatedMeter(baseLoad, variance uint64) *SimulatedMeter {
	return &SimulatedMeter{
		BaseLoad: baseLoad,
		Variance: variance,
	}
}

// Generate creates a new reading by combining the meter's base load
// with a randomized fluctuation, simulating realistic usage spikes.
func (meter *SimulatedMeter) Generate() Reading {
	fluctuation := uint64(rand.Int63n(int64(meter.Variance)))

	return Reading{
		Timestamp:   time.Now().Unix(),
		Consumption: meter.BaseLoad + fluctuation,
	}
}
