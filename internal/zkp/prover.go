package zkp

import (
	"fmt"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"os"
)

type Engine struct {
	ProvingKey               groth16.ProvingKey
	VerifyingKey             groth16.VerifyingKey
	CompiledConstraintSystem constraint.ConstraintSystem
}

func Setup() (*Engine, error) {
	var circuit RangeProofCircuit

	compiledConstraintSystem, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)

	if err != nil {
		return nil, fmt.Errorf("an error occured during compiling circuit: %w", err)
	}

	provingKey, verifyingKey, err := groth16.Setup(compiledConstraintSystem)
	if err != nil {
		return nil, fmt.Errorf("an error occured during generating keys: %w", err)
	}

	pkFile, err := os.Create("proving.key")
	if err != nil {
		return nil, fmt.Errorf("failed to create proving.key file: %w", err)
	}

	_, err = provingKey.WriteTo(pkFile)
	if err != nil {
		closeErr := pkFile.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("failed to write proving key: %v (also failed to close file: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to write proving key: %w", err)
	}

	if err := pkFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close proving key file: %w", err)
	}
	fmt.Println("[SETUP] Successfully saved proving.key")

	vkFile, err := os.Create("verifying.key")
	if err != nil {
		return nil, fmt.Errorf("failed to create verifying.key file: %w", err)
	}

	_, err = verifyingKey.WriteTo(vkFile)
	if err != nil {
		closeErr := vkFile.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("failed to write verifying key: %v (also failed to close file: %v)", err, closeErr)
		}
		return nil, fmt.Errorf("failed to write verifying key: %w", err)
	}

	if err := vkFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close verifying key file: %w", err)
	}
	fmt.Println("[SETUP] Successfully saved verifying.key")

	return &Engine{
		ProvingKey:               provingKey,
		VerifyingKey:             verifyingKey,
		CompiledConstraintSystem: compiledConstraintSystem,
	}, nil
}

func (engine *Engine) GenerateProof(consumption, maxLimit uint64, meterID, timestamp uint64) (groth16.Proof, error) {
	assignment := &RangeProofCircuit{
		Consumption: consumption,
		MaxLimit:    maxLimit,
		MeterID:     meterID,
		Timestamp:   timestamp,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("An error occured during generating witness: %w", err)
	}

	proof, err := groth16.Prove(engine.CompiledConstraintSystem, engine.ProvingKey, witness)

	if err != nil {
		return nil, fmt.Errorf("An error occured during generating proof: %w", err)
	}

	return proof, nil
}
