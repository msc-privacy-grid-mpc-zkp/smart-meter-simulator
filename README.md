# ⚡ Smart Meter Edge Simulator (ZKP Privacy-Preserving Grid)

*[Read in English](#english) | [Čitaj na Srpskom](#srpski)*

---

<a name="english"></a>
## 🇬🇧 English

### Overview
The **Smart Meter Edge Simulator** is a core component of a privacy-preserving smart grid architecture. Written in Go, it simulates multiple edge devices (smart meters) that generate realistic power consumption data. Instead of sending raw, sensitive data to the cloud, each meter computes a **Zero-Knowledge Proof (ZKP)** using the `gnark` library (Groth16 over BN254). 

This proof cryptographically guarantees that the power consumption is strictly positive and below the physical hardware limit, without ever revealing the actual consumption value to the network. The proofs are then asynchronously transmitted to the MPC Cloud Aggregator.

### Key Features
* **Edge ZKP Generation:** Utilizes `gnark` to compile R1CS circuits and generate Groth16 proofs locally on the edge device.
* **High-Performance Concurrency:** Implements a robust **Worker Pool** pattern with goroutines, allowing a single machine to simulate hundreds of smart meters simultaneously without CPU bottlenecking.
* **Robust Networking:** Features a fault-tolerant HTTP client with strict timeouts and proper resource cleanup.
* **Fully Configurable:** Easily tweak the number of meters, worker threads, load variance, and network endpoints via a central `config.yaml` file.

### Prerequisites
* [Go 1.20+](https://go.dev/dl/)
* Git

### Getting Started

1. **Clone the repository:**
   ```bash
   git clone [https://github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator.git](https://github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator.git)
   cd smart-meter-simulator

3. **Run the simulation:**
   ```bash
   go run cmd/simulator/main.go
