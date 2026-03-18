package network

import (
	"bytes"
	"fmt"

	"github.com/consensys/gnark/backend/groth16"
)

type ProofPayload struct {
	MeterID    string `json:"meter_id"`
	Timestamp  int64  `json:"timestamp"`
	MeterShare int64  `json:"meter_share"`
	Proof      []byte `json:"proof"`
}

func SerializeProof(proof groth16.Proof) ([]byte, error) {
	var buf bytes.Buffer

	_, err := proof.WriteTo(&buf)
	if err != nil {
		return nil, fmt.Errorf("error during proof serialization: %w", err)
	}

	return buf.Bytes(), nil
}
