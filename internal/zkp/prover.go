package zkp

import (
	"fmt"
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

func Setup() (*Engine, error) {
	var circuit RangeProofCircuit

	compiledConstraintSystem, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)

	if err != nil {
		return nil, fmt.Errorf("An error occured during compiling circuit: %w", err)
	}

	provingKey, verifyingKey, err := groth16.Setup(compiledConstraintSystem)
	if err != nil {
		return nil, fmt.Errorf("An error occured during generating keys: %w", err)
	}

	return &Engine{
		ProvingKey:               provingKey,
		VerifyingKey:             verifyingKey,
		CompiledConstraintSystem: compiledConstraintSystem,
	}, nil
}

func (engine *Engine) GenerateProof(consumption, maxLimit uint64) (groth16.Proof, error) {
	assignment := &RangeProofCircuit{
		Consumption: consumption,
		MaxLimit:    maxLimit,
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
