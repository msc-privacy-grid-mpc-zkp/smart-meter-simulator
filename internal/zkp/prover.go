package zkp

import (
	"fmt"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

type Engine struct {
	ProvingKey               groth16.ProvingKey
	VerifyingKey             groth16.VerifyingKey
	CompiledConstraintSystem constraint.ConstraintSystem
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Setup() (*Engine, error) {
	var circuit RangeProofCircuit

	compiledConstraintSystem, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		return nil, fmt.Errorf("an error occured during compiling circuit: %w", err)
	}

	var provingKey groth16.ProvingKey
	var verifyingKey groth16.VerifyingKey

	if fileExists("proving.key") && fileExists("verifying.key") {
		fmt.Println("[SETUP] Found existing keys on disk. Loading...")

		provingKey = groth16.NewProvingKey(ecc.BN254)
		pkFile, err := os.Open("proving.key")
		if err != nil {
			return nil, fmt.Errorf("failed to open proving.key: %w", err)
		}
		defer pkFile.Close()
		if _, err := provingKey.ReadFrom(pkFile); err != nil {
			return nil, fmt.Errorf("failed to read proving key: %w", err)
		}

		verifyingKey = groth16.NewVerifyingKey(ecc.BN254)
		vkFile, err := os.Open("verifying.key")
		if err != nil {
			return nil, fmt.Errorf("failed to open verifying.key: %w", err)
		}
		defer vkFile.Close()
		if _, err := verifyingKey.ReadFrom(vkFile); err != nil {
			return nil, fmt.Errorf("failed to read verifying key: %w", err)
		}

		fmt.Println("[SETUP] Keys loaded successfully!")

	} else {
		fmt.Println("[SETUP] Keys not found. Generating new Trusted Setup (this may take a moment)...")

		provingKey, verifyingKey, err = groth16.Setup(compiledConstraintSystem)
		if err != nil {
			return nil, fmt.Errorf("an error occured during generating keys: %w", err)
		}

		pkOut, err := os.Create("proving.key")
		if err != nil {
			return nil, fmt.Errorf("failed to create proving.key file: %w", err)
		}
		defer pkOut.Close()
		if _, err := provingKey.WriteTo(pkOut); err != nil {
			return nil, fmt.Errorf("failed to write proving key: %w", err)
		}

		vkOut, err := os.Create("verifying.key")
		if err != nil {
			return nil, fmt.Errorf("failed to create verifying.key file: %w", err)
		}
		defer vkOut.Close()
		if _, err := verifyingKey.WriteTo(vkOut); err != nil {
			return nil, fmt.Errorf("failed to write verifying key: %w", err)
		}

		fmt.Println("[SETUP] Successfully generated and saved keys to disk!")
	}

	return &Engine{
		ProvingKey:               provingKey,
		VerifyingKey:             verifyingKey,
		CompiledConstraintSystem: compiledConstraintSystem,
	}, nil
}

func (engine *Engine) GenerateProof(consumption, maxLimit, meterID, timestamp uint64) (groth16.Proof, error) {
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
