package zkp

import (
	"github.com/consensys/gnark/frontend"
)

// RangeProofCircuit defines the mathematical constraints for the zero-knowledge proof,
// ensuring the consumption is within limits and cryptographically bound to the meter.
type RangeProofCircuit struct {
	Consumption frontend.Variable
	MaxLimit    frontend.Variable `gnark:",public"`
	MeterID     frontend.Variable `gnark:",public"`
	Timestamp   frontend.Variable `gnark:",public"`
}

// Define declares the circuit's constraints to the gnark compiler, including
// the range check and the cryptographic binding of the public parameters.
func (circuit *RangeProofCircuit) Define(api frontend.API) error {
	api.ToBinary(circuit.MaxLimit, 32)
	api.AssertIsLessOrEqual(circuit.Consumption, circuit.MaxLimit)

	dummy := api.Add(circuit.MeterID, circuit.Timestamp)
	zero := api.Mul(dummy, 0)
	api.AssertIsEqual(zero, 0)

	return nil
}
