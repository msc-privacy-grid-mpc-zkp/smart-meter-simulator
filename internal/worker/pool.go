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
	zkpEngine  *zkp.Engine
	clients    []*network.CloudClient
}

// NewPool initializes a new worker pool
func NewPool(workerSize int, queueSize int, maxLimit uint64, zkpEngine *zkp.Engine, clients []*network.CloudClient) *Pool {
	return &Pool{
		Jobs:       make(chan Job, queueSize),
		wg:         &sync.WaitGroup{},
		workerSize: workerSize,
		maxLimit:   maxLimit,
		zkpEngine:  zkpEngine,
		clients:    clients,
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
	numServers := len(p.clients)

	for job := range p.Jobs {
		// 1. ZKP Generation
		proof, err := p.zkpEngine.GenerateProof(job.Reading.Consumption, p.maxLimit)
		if err != nil {
			log.Printf("[Worker %d] ZKP Error for %s: %v", id, job.MeterID, err)
			continue
		}
		proofBytes, _ := network.SerializeProof(proof)
		actualConsumption := int64(job.Reading.Consumption)

		// 2. Multi-Party Secret Sharing (N-shares)
		shares := make([]int64, numServers)
		var sumOfShares int64 = 0

		for i := 0; i < numServers-1; i++ {
			// Koristimo Int63 ali skalirano da izbegnemo teoretski overflow pri sumiranju
			shares[i] = rand.Int63() / int64(numServers)
			sumOfShares += shares[i]
		}
		shares[numServers-1] = actualConsumption - sumOfShares

		// 3. Dispatching to N Servers
		allSuccess := true
		for i, client := range p.clients {
			payload := network.ProofPayload{
				MeterID:    job.MeterID,
				Timestamp:  job.Reading.Timestamp,
				MeterShare: shares[i],
				Proof:      proofBytes,
			}

			// Pretpostavljamo da SendProof vraća error ako HTTP status nije 200 OK
			if err := client.SendProof(payload); err != nil {
				log.Printf("[Worker %d] Failed to send share to Server %d: %v", id, i, err)
				allSuccess = false
			}
		}

		// 4. Final Logging - Ovo je ono što ti nedostaje!
		if allSuccess {
			fmt.Printf("[Worker %d] ✅ Successfully dispatched %d shares for %s (Actual: %dW)\n",
				id, numServers, job.MeterID, actualConsumption)
		}
	}
}
