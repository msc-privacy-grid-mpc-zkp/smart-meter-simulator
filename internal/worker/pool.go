package worker

import (
	"fmt"
	"log"
	"sync"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/utils"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

// Job represents a single unit of work for the worker pool, containing
// the meter identifier and its latest consumption reading.
type Job struct {
	MeterID string
	Reading meter.Reading
}

// Pool manages a group of concurrent workers that process incoming meter readings.
// It handles Zero-Knowledge Proof generation, Multi-Party Computation share splitting,
// and network dispatch to the aggregator nodes.
type Pool struct {
	Jobs       chan Job
	wg         *sync.WaitGroup
	workerSize int
	maxLimit   uint64
	zkpEngine  *zkp.Engine
	clients    []*network.Client
}

// NewPool initializes a new worker pool with the specified concurrency size,
// job queue capacity, cryptographic engine, and network clients.
func NewPool(workerSize, queueSize int, maxLimit uint64, zkpEngine *zkp.Engine, clients []*network.Client) *Pool {
	return &Pool{
		Jobs:       make(chan Job, queueSize),
		wg:         &sync.WaitGroup{},
		workerSize: workerSize,
		maxLimit:   maxLimit,
		zkpEngine:  zkpEngine,
		clients:    clients,
	}
}

// Start launches the worker goroutines, actively listening for incoming jobs.
func (p *Pool) Start() {
	for w := 1; w <= p.workerSize; w++ {
		p.wg.Add(1)
		go p.worker(w)
	}
}

// Wait blocks until all workers in the pool have finished their current jobs
// and exited. This should be called after closing the Jobs channel.
func (p *Pool) Wait() {
	p.wg.Wait()
}

// worker processes jobs from the queue: generates ZKP, splits data into MPC shares,
// and concurrently transmits the payloads to all aggregator nodes.
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	numServers := len(p.clients)

	for job := range p.Jobs {
		numericMeterID := crypto.HashStringToUint64(job.MeterID)
		proof, err := p.zkpEngine.GenerateProof(
			job.Reading.Consumption,
			p.maxLimit,
			numericMeterID,
			uint64(job.Reading.Timestamp),
		)
		if err != nil {
			log.Printf("[Worker %d] ZKP Error for %s: %v\n", id, job.MeterID, err)
			continue
		}

		proofBytes, err := network.SerializeProof(proof)
		if err != nil {
			log.Printf("[Worker %d] Serialization Error for %s: %v\n", id, job.MeterID, err)
			continue
		}

		// 2. MPC Share Splitting
		actualConsumption := int64(job.Reading.Consumption)
		shares := make([]int64, numServers)
		var sumOfShares int64 = 0

		for i := 0; i < numServers-1; i++ {
			shares[i] = crypto.SecureRandomInt64()
			sumOfShares += shares[i]
		}
		shares[numServers-1] = actualConsumption - sumOfShares

		var sendWg sync.WaitGroup
		var mu sync.Mutex
		allSuccess := true

		for i, client := range p.clients {
			sendWg.Add(1)

			go func(serverIdx int, cl *network.Client, share int64) {
				defer sendWg.Done()

				payload := network.ProofPayload{
					MeterID:    job.MeterID,
					Timestamp:  job.Reading.Timestamp,
					MeterShare: share,
					Proof:      proofBytes,
				}

				if err := cl.SendProof(payload); err != nil {
					log.Printf("[Worker %d] Server %d Unreachable: %v\n", id, serverIdx, err)
					mu.Lock()
					allSuccess = false
					mu.Unlock()
				}
			}(i, client, shares[i])
		}

		sendWg.Wait()

		if allSuccess {
			fmt.Printf("[Worker %d] ✅ ZKP+MPC Dispatched | Meter: %s | Nodes: %d | Val: %dW\n",
				id, job.MeterID, numServers, actualConsumption)
		}
	}
}
