package worker

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"

	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/meter"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/network"
	"github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator/internal/zkp"
)

type Job struct {
	MeterID string
	Reading meter.Reading
}

type Pool struct {
	Jobs       chan Job
	wg         *sync.WaitGroup
	workerSize int
	maxLimit   uint64
	zkpEngine  *zkp.Engine
	clients    []*network.CloudClient
}

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

func (p *Pool) Start() {
	for w := 1; w <= p.workerSize; w++ {
		p.wg.Add(1)
		go p.worker(w)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	numServers := len(p.clients)

	for job := range p.Jobs {
		proof, err := p.zkpEngine.GenerateProof(job.Reading.Consumption, p.maxLimit)
		if err != nil {
			log.Printf("[Worker %d] ZKP Error for %s: %v", id, job.MeterID, err)
			continue
		}
		proofBytes, _ := network.SerializeProof(proof)

		actualConsumption := int64(job.Reading.Consumption)
		shares := make([]int64, numServers)
		var sumOfShares int64 = 0

		for i := 0; i < numServers-1; i++ {
			// Generiši čist random broj (masku)
			shares[i] = p.secureRandomInt64()
			sumOfShares += shares[i]
		}
		// Poslednji server dobija razliku - ovo osigurava da je suma shares == actualConsumption
		shares[numServers-1] = actualConsumption - sumOfShares

		allSuccess := true
		for i, client := range p.clients {
			payload := network.ProofPayload{
				MeterID:    job.MeterID,
				Timestamp:  job.Reading.Timestamp,
				MeterShare: shares[i],
				Proof:      proofBytes,
			}
			if err := client.SendProof(payload); err != nil {
				log.Printf("[Worker %d] Server %d Unreachable: %v", id, i, err)
				allSuccess = false
			}
		}

		if allSuccess {
			fmt.Printf("[Worker %d] ✅ ZKP+MPC Dispatched | Meter: %s | Nodes: %d | Val: %dW\n",
				id, job.MeterID, numServers, actualConsumption)
		}
	}
}

// secureRandomInt64 generates a cryptographically secure random int64
func (p *Pool) secureRandomInt64() int64 {
	maxNum := big.NewInt(1 << 40)
	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		// Fallback na 0 u slučaju kritične greške OS entropije
		return 0
	}
	return n.Int64()
}
