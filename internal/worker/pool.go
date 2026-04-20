package worker

import (
	"crypto/rand"
	"fmt"
	"hash/fnv"
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
		// 1. ZKP Generisanje (CPU bound)
		numericMeterID := stringToUint64(job.MeterID)
		proof, err := p.zkpEngine.GenerateProof(
			job.Reading.Consumption,
			p.maxLimit,
			numericMeterID,
			uint64(job.Reading.Timestamp), // Timestamp претварамо у uint64
		)

		if err != nil {
			log.Printf("[Worker %d] ZKP Error for %s: %v", id, job.MeterID, err)
			continue
		}
		proofBytes, _ := network.SerializeProof(proof)

		// 2. Kreiranje MPC delova
		actualConsumption := int64(job.Reading.Consumption)
		shares := make([]int64, numServers)
		var sumOfShares int64 = 0

		for i := 0; i < numServers-1; i++ {
			shares[i] = p.secureRandomInt64()
			sumOfShares += shares[i]
		}
		shares[numServers-1] = actualConsumption - sumOfShares

		// 3. PARALELNO SLANJE (I/O bound)
		var sendWg sync.WaitGroup
		var mu sync.Mutex // Za zaštitu allSuccess varijable
		allSuccess := true

		for i, client := range p.clients {
			sendWg.Add(1)

			// Pokrećemo anonimnu gorutinu za svaki server
			go func(serverIdx int, cl *network.CloudClient, share int64) {
				defer sendWg.Done()

				payload := network.ProofPayload{
					MeterID:    job.MeterID,
					Timestamp:  job.Reading.Timestamp,
					MeterShare: share,
					Proof:      proofBytes,
				}

				if err := cl.SendProof(payload); err != nil {
					log.Printf("[Worker %d] Server %d Unreachable: %v", id, serverIdx, err)

					mu.Lock()
					allSuccess = false
					mu.Unlock()
				}
			}(i, client, shares[i]) // Prosleđujemo promenljive u gorutinu
		}

		// Čekamo da sva 3 servera odgovore (umesto da čekamo jedan po jedan)
		sendWg.Wait()

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

func stringToUint64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
