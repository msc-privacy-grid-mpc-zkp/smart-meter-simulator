package network

import (
	"bytes"
	"fmt"

	"github.com/consensys/gnark/backend/groth16"
)

// ProofPayload represents the data structure transmitted from the edge meter to the aggregator.
type ProofPayload struct {
	MeterID    string `json:"meter_id"`
	Timestamp  int64  `json:"timestamp"`
	MeterShare int64  `json:"meter_share"`
	Proof      []byte `json:"proof"`
}

// SerializeProof converts a Groth16 proof into a byte slice for network transmission.
func SerializeProof(proof groth16.Proof) ([]byte, error) {
	var buf bytes.Buffer

	if _, err := proof.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("error during proof serialization: %w", err)
	}

	return buf.Bytes(), nil
}
