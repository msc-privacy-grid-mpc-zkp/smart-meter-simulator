package worker

import (
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/utils"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

// tamperedCount tracks how many payloads have been maliciously tampered.
// Used for Test 1.2: Public Input Tampering with configurable meter count.
var tamperedCount atomic.Int32

// ResetTamperedCount resets the tamperedCount to 0.
// This is called at the beginning of each synchronization cycle to ensure
// that exactly MaliciousTamperCount meters are tampered in every cycle.
func ResetTamperedCount() {
	tamperedCount.Store(0)
}

// poisonedCount tracks how many payloads have been poisoned with invalid shares.
// Used for Test 2.1: Data Poisoning with configurable meter count.
var poisonedCount atomic.Int32

// ResetPoisonedCount resets the poisonedCount to 0.
// This is called at the beginning of each synchronization cycle to ensure
// that exactly MaliciousPoisoningCount meters are poisoned in every cycle.
func ResetPoisonedCount() {
	poisonedCount.Store(0)
}

// overflowCount tracks how many overflow attack cycles have been triggered.
// Used for Test 2.2: Integer Overflow Attempt with configurable cycle count.
var overflowCount atomic.Int32

// ResetOverflowCount resets the overflowCount to 0.
// This is called at the beginning of each synchronization cycle.
func ResetOverflowCount() {
	overflowCount.Store(0)
}

// GetOverflowCount returns the current overflow cycle count.
func GetOverflowCount() int32 {
	return overflowCount.Load()
}

// IncrementOverflowCount increments the overflow cycle counter.
func IncrementOverflowCount() {
	overflowCount.Add(1)
}

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
	Jobs                    chan Job
	wg                      *sync.WaitGroup
	workerSize              int
	maxLimit                uint64
	zkpEngine               *zkp.Engine
	clients                 []*network.Client
	maliciousReplay         bool
	maliciousTamperCount    int
	maliciousNoise          bool
	maliciousPoisoningCount int
	maliciousOverflowCount  int
	maliciousOverflowMeters int
}

// NewPool initializes a new worker pool with the specified concurrency size,
// job queue capacity, cryptographic engine, network clients, and red team flags.
func NewPool(workerSize, queueSize int, maxLimit uint64, zkpEngine *zkp.Engine, clients []*network.Client, maliciousReplay bool, maliciousTamperCount int, maliciousNoise bool, maliciousPoisoningCount int, maliciousOverflowCount int, maliciousOverflowMeters int) *Pool {
	return &Pool{
		Jobs:                    make(chan Job, queueSize),
		wg:                      &sync.WaitGroup{},
		workerSize:              workerSize,
		maxLimit:                maxLimit,
		zkpEngine:               zkpEngine,
		clients:                 clients,
		maliciousReplay:         maliciousReplay,
		maliciousTamperCount:    maliciousTamperCount,
		maliciousNoise:          maliciousNoise,
		maliciousPoisoningCount: maliciousPoisoningCount,
		maliciousOverflowCount:  maliciousOverflowCount,
		maliciousOverflowMeters: maliciousOverflowMeters,
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

		// RED TEAM: Test 2.1 - Data Poisoning / Invalid Shares
		// Determine if this meter should be poisoned
		shouldPoison := false
		if p.maliciousPoisoningCount > 0 && poisonedCount.Load() < int32(p.maliciousPoisoningCount) {
			if poisonedCount.Add(1) <= int32(p.maliciousPoisoningCount) {
				shouldPoison = true
			}
		}

		// Generate ZKP proof for the LEGITIMATE consumption (to bypass ZKP verification)
		proof, commitment, err := p.zkpEngine.GenerateProof(
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

		// RED TEAM: Test 1.3 - Random Noise (Invalid Proof)
		// If enabled, replace the valid proof with random garbage of the same length
		if p.maliciousNoise {
			randomNoise := make([]byte, len(proofBytes))
			if _, err := rand.Read(randomNoise); err != nil {
				log.Printf("[Worker %d] Failed to generate random noise for %s: %v\n", id, job.MeterID, err)
				continue
			}
			proofBytes = randomNoise
			log.Printf("[Worker %d] 🔴 RED TEAM: Injecting random noise (invalid proof) for meter %s\n", id, job.MeterID)
		}

		// 2. MPC Share Splitting
		// Originalna potrošnja sada ostaje uint64 (nema cast-ovanja)
		actualConsumption := job.Reading.Consumption

		// RED TEAM: Test 2.1 - Data Poisoning / Invalid Shares
		// If this meter is poisoned, use an extreme outlier value for share generation
		// while keeping the ZKP proof valid for the legitimate consumption
		consumptionForShares := actualConsumption
		if shouldPoison {
			// Use an extreme outlier value (999,999,999 Watts)
			consumptionForShares = 999_999_999
			log.Printf("[Worker %d] 🔴 RED TEAM: Poisoning meter %s with extreme value %d (ZKP proof valid for %d) [%d/%d]\n",
				id, job.MeterID, consumptionForShares, actualConsumption, poisonedCount.Load(), p.maliciousPoisoningCount)
		}

		shares := make([]uint64, numServers)
		var sumOfShares uint64 = 0

		for i := 0; i < numServers-1; i++ {
			shares[i] = crypto.SecureRandomUint64() // Pozivamo novu funkciju
			sumOfShares += shares[i]
		}
		// Oduzimanje se sada automatski odvija po modulu 2^64
		shares[numServers-1] = consumptionForShares - sumOfShares

		var sendWg sync.WaitGroup
		var mu sync.Mutex
		allSuccess := true

		for i, client := range p.clients {
			sendWg.Add(1)

			// Promijenjen tip parametra share u uint64
			go func(serverIdx int, cl *network.Client, share uint64) {
				defer sendWg.Done()

				payload := network.ProofPayload{
					MeterID:    job.MeterID,
					Timestamp:  job.Reading.Timestamp,
					MeterShare: share,
					Proof:      proofBytes,
					Commitment: commitment,
				}

				// RED TEAM: Test 1.2 - Public Input Tampering
				// If enabled, mutate the MeterID in the payload ONLY (proof remains valid for original meter)
				// Only tamper up to the configured number of meters
				if p.maliciousTamperCount > 0 && tamperedCount.Load() < int32(p.maliciousTamperCount) {
					if tamperedCount.Add(1) <= int32(p.maliciousTamperCount) {
						payload.MeterID = payload.MeterID + "-FAKE"
						log.Printf("[Worker %d] 🔴 RED TEAM: Tampering MeterID to %s (proof still bound to original meter) [%d/%d]\n",
							id, payload.MeterID, tamperedCount.Load(), p.maliciousTamperCount)
					}
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
			// Check if this is an overflow attack meter
			isOverflowMeter := len(job.MeterID) > 14 && job.MeterID[:14] == "meter-OVERFLOW"
			if isOverflowMeter {
				fmt.Printf("[Worker %d] 🔴 OVERFLOW ATTACK | Meter: %s | Nodes: %d | Val: %dW (at physical limit)\n",
					id, job.MeterID, numServers, actualConsumption)
			} else {
				fmt.Printf("[Worker %d] ✅ ZKP+MPC Dispatched | Meter: %s | Nodes: %d | Val: %dW\n",
					id, job.MeterID, numServers, actualConsumption)
			}
		}

		// RED TEAM: Test 1.1 - Replay Attack Simulation
		// If enabled, immediately resend the EXACT same payload without regenerating proof or updating timestamp
		if p.maliciousReplay && allSuccess {
			log.Printf("[Worker %d] 🔴 RED TEAM: Initiating Replay Attack for meter %s\n", id, job.MeterID)

			var replayWg sync.WaitGroup
			var replayMu sync.Mutex
			replaySuccess := true

			for i, client := range p.clients {
				replayWg.Add(1)

				// Send the IDENTICAL payload again (no proof regeneration, no timestamp update)
				go func(serverIdx int, cl *network.Client, share uint64) {
					defer replayWg.Done()

					// Clone the exact same payload object
					replayPayload := network.ProofPayload{
						MeterID:    job.MeterID,
						Timestamp:  job.Reading.Timestamp,
						MeterShare: share,
						Proof:      proofBytes,
						Commitment: commitment,
					}

					if err := cl.SendProof(replayPayload); err != nil {
						log.Printf("[Worker %d] Replay Attack - Server %d Unreachable: %v\n", id, serverIdx, err)
						replayMu.Lock()
						replaySuccess = false
						replayMu.Unlock()
					}
				}(i, client, shares[i])
			}

			replayWg.Wait()

			if replaySuccess {
				fmt.Printf("[Worker %d] 🔴 REPLAY ATTACK SUCCESSFUL | Meter: %s | Duplicate payload sent to all nodes\n",
					id, job.MeterID)
			}
		}
	}
}
