package zkp

import (
	"fmt"
	"log"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

const (
	provingKeyPath   = "keys/proving.key"
	verifyingKeyPath = "keys/verifying.key"
)

// Engine encapsulates the Groth16 cryptographic keys and the compiled
// constraint system required for generating Zero-Knowledge proofs.
type Engine struct {
	ProvingKey               groth16.ProvingKey
	VerifyingKey             groth16.VerifyingKey
	CompiledConstraintSystem constraint.ConstraintSystem
}

// Setup initializes the ZKP Engine by compiling the RangeProofCircuit.
// It attempts to load existing proving and verifying keys from disk to bypass
// the expensive Trusted Setup phase. If keys are not found, it generates new
// ones and persists them to the file system for future executions.
func Setup() (*Engine, error) {
	var circuit RangeProofCircuit

	compiledConstraintSystem, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		return nil, fmt.Errorf("an error occurred during compiling circuit: %w", err)
	}

	var provingKey groth16.ProvingKey
	var verifyingKey groth16.VerifyingKey

	if fileExists(provingKeyPath) && fileExists(verifyingKeyPath) {
		provingKey, verifyingKey, err = loadKeysFromDisk()
		if err != nil {
			return nil, fmt.Errorf("failed to load existing keys: %w", err)
		}
	} else {
		provingKey, verifyingKey, err = generateAndSaveKeys(compiledConstraintSystem)
		if err != nil {
			return nil, fmt.Errorf("failed to generate new keys: %w", err)
		}
	}

	return &Engine{
		ProvingKey:               provingKey,
		VerifyingKey:             verifyingKey,
		CompiledConstraintSystem: compiledConstraintSystem,
	}, nil
}

// GenerateProof computes a zk-SNARK proof demonstrating that a secret consumption
// value is less than or equal to the public maxLimit, while cryptographically
// binding the proof to a specific meterID and timestamp to prevent replay attacks.
func (engine *Engine) GenerateProof(consumption, maxLimit, meterID, timestamp uint64) (groth16.Proof, error) {
	assignment := &RangeProofCircuit{
		Consumption: consumption,
		MaxLimit:    maxLimit,
		MeterID:     meterID,
		Timestamp:   timestamp,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("an error occurred during generating witness: %w", err)
	}

	proof, err := groth16.Prove(engine.CompiledConstraintSystem, engine.ProvingKey, witness)
	if err != nil {
		return nil, fmt.Errorf("an error occurred during generating proof: %w", err)
	}

	return proof, nil
}

// --- Private Helper Functions ---

// loadKeysFromDisk reads the previously generated proving and verifying keys
// from the local file system and deserializes them into memory.
func loadKeysFromDisk() (groth16.ProvingKey, groth16.VerifyingKey, error) {
	log.Println("[SETUP] Found existing keys on disk. Loading...")

	provingKey := groth16.NewProvingKey(ecc.BN254)
	pkFile, err := os.Open(provingKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %w", provingKeyPath, err)
	}
	defer func() {
		if err := pkFile.Close(); err != nil {
			log.Printf("[WARNING] Failed to close %s after reading: %v\n", provingKeyPath, err)
		}
	}()
	if _, err := provingKey.ReadFrom(pkFile); err != nil {
		return nil, nil, fmt.Errorf("failed to read proving key: %w", err)
	}

	verifyingKey := groth16.NewVerifyingKey(ecc.BN254)
	vkFile, err := os.Open(verifyingKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open %s: %w", verifyingKeyPath, err)
	}
	defer func() {
		if err := vkFile.Close(); err != nil {
			log.Printf("[WARNING] Failed to close %s after reading: %v\n", verifyingKeyPath, err)
		}
	}()
	if _, err := verifyingKey.ReadFrom(vkFile); err != nil {
		return nil, nil, fmt.Errorf("failed to read verifying key: %w", err)
	}

	log.Println("[SETUP] Keys loaded successfully!")
	return provingKey, verifyingKey, nil
}

// generateAndSaveKeys executes the Groth16 Trusted Setup to create new
// cryptographic parameters for the given constraint system, and saves them
// to disk to accelerate future startups.
func generateAndSaveKeys(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
	log.Println("[SETUP] Keys not found. Generating new Trusted Setup (this may take a moment)...")

	provingKey, verifyingKey, err := groth16.Setup(ccs)
	if err != nil {
		return nil, nil, fmt.Errorf("an error occurred during setup: %w", err)
	}

	pkOut, err := os.Create(provingKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create %s: %w", provingKeyPath, err)
	}
	defer func() {
		if err := pkOut.Close(); err != nil {
			log.Printf("[WARNING] Failed to close %s after writing: %v\n", provingKeyPath, err)
		}
	}()
	if _, err := provingKey.WriteTo(pkOut); err != nil {
		return nil, nil, fmt.Errorf("failed to write proving key: %w", err)
	}

	vkOut, err := os.Create(verifyingKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create %s: %w", verifyingKeyPath, err)
	}
	defer func() {
		if err := vkOut.Close(); err != nil {
			log.Printf("[WARNING] Failed to close %s after writing: %v\n", verifyingKeyPath, err)
		}
	}()
	if _, err := verifyingKey.WriteTo(vkOut); err != nil {
		return nil, nil, fmt.Errorf("failed to write verifying key: %w", err)
	}

	log.Println("[SETUP] Successfully generated and saved keys to disk!")
	return provingKey, verifyingKey, nil
}

// fileExists is a helper function that checks if a file exists and is not a directory.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
