package meter

import (
	"math/rand"
	"time"
)

type Reading struct {
	Timestamp   int64
	Consumption uint64
}

type Generator interface {
	Generate() Reading
}

type SimulatedMeter struct {
	BaseLoad uint64
	Variance uint64
}

func NewSimulatedMeter(baseLoad, variance uint64) *SimulatedMeter {
	return &SimulatedMeter{BaseLoad: baseLoad, Variance: variance}
}

func (meter *SimulatedMeter) Generate() Reading {
	fluctuation := uint64(rand.Int63n(int64(meter.Variance)))
	return Reading{
		Timestamp:   time.Now().Unix(),
		Consumption: meter.BaseLoad + fluctuation,
	}
}
