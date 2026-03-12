package worker

import (
	"fmt"
	"log"
	"math/rand"
	"sync"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

// Job represents a single processing task for a worker
type Job struct {
	MeterID string
	Reading meter.Reading
}

// Pool encapsulates the worker pool logic and dependencies
type Pool struct {
	Jobs       chan Job
	wg         *sync.WaitGroup
	workerSize int
	maxLimit   uint64
	zkpEngine  *zkp.Engine // Napomena: Prilagodi tip ako se tvoj struct zove drugačije
	clientA    *network.CloudClient
	clientB    *network.CloudClient
}

// NewPool initializes a new worker pool
func NewPool(workerSize int, queueSize int, maxLimit uint64, zkpEngine *zkp.Engine, clientA, clientB *network.CloudClient) *Pool {
	return &Pool{
		Jobs:       make(chan Job, queueSize),
		wg:         &sync.WaitGroup{},
		workerSize: workerSize,
		maxLimit:   maxLimit,
		zkpEngine:  zkpEngine,
		clientA:    clientA,
		clientB:    clientB,
	}
}

// Start spins up the configured number of worker goroutines
func (p *Pool) Start() {
	for w := 1; w <= p.workerSize; w++ {
		p.wg.Add(1)
		go p.worker(w)
	}
}

// worker is the actual goroutine function processing the jobs
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for job := range p.Jobs {
		proof, err := p.zkpEngine.GenerateProof(job.Reading.Consumption, p.maxLimit)
		if err != nil {
			log.Printf("[Worker %d] Error generating proof for %s: %v\n", id, job.MeterID, err)
			continue
		}
		proofBytes, _ := network.SerializeProof(proof)

		actualConsumption := int64(job.Reading.Consumption)

		share1 := rand.Int63()
		share2 := actualConsumption - share1

		payloadA := network.ProofPayload{
			MeterID:    job.MeterID,
			Timestamp:  job.Reading.Timestamp,
			MeterShare: share1,
			Proof:      proofBytes,
		}

		payloadB := network.ProofPayload{
			MeterID:    job.MeterID,
			Timestamp:  job.Reading.Timestamp,
			MeterShare: share2,
			Proof:      proofBytes,
		}

		errA := p.clientA.SendProof(payloadA)
		errB := p.clientB.SendProof(payloadB)

		if errA != nil || errB != nil {
			log.Printf("[Worker %d] Network error for %s (A: %v, B: %v)\n", id, job.MeterID, errA, errB)
		} else {
			fmt.Printf("[Worker %d] ZKP + MPC sent for %s (Actual %dW masked as %d and %d)\n",
				id, job.MeterID, actualConsumption, share1, share2)
		}
	}
}
