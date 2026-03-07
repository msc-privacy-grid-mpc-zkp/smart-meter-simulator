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
   git clone https://github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator.git
   cd smart-meter-simulator

1. **Install dependencies:**
   ```bash
   go mod tidy

3. **Run the simulation:**
   ```bash
   go run cmd/simulator/main.go

<a name="srpski"></a>
## 🇷🇸 Srpski
### Pregled arhitekture
**Smart Meter Edge Simulator** je osnovna komponenta pametne elektro-mreže dizajnirane sa fokusom na zaštitu privatnosti korisnika. Napisan u Go programskom jeziku, ovaj sistem simulira višestruke edge uređaje (pametna brojila) koji generišu realistične podatke o potrošnji električne energije.

Umesto da šalje sirove, osetljive podatke u cloud, svako brojilo računa **Zero-Knowledge dokaz (ZKP)** koristeći gnark biblioteku (Groth16 nad BN254 krivom). Ovaj dokaz kriptografski garantuje da je potrošnja striktno pozitivna i manja od maksimalnog hardverskog limita priključka, bez otkrivanja stvarne vrednosti potrošnje. Dokazi se zatim asinhrono šalju ka MPC Cloud Agregatoru.

### Ključne funkcionalnosti
* **ZKP na ivici mreže (Edge)**: Korišćenje `gnark` biblioteke za prevođenje R1CS kola i lokalno generisanje Groth16 dokaza.

* **Visoke performanse (Concurrency)**: Implementiran **Worker Pool** obrazac sa Go rutinama, što omogućava da jedna mašina simulira stotine brojila istovremeno bez zagušenja procesora.

* **Robustna mreža**: Otporan HTTP klijent sa striktnim timeout-om i pravilnim oslobađanjem sistemskih resursa.

* **Konfigurabilnost**: Lako podešavanje broja brojila, broja paralelnih procesa, varijanse potrošnje i mrežnih parametara kroz jedan `config.yaml` fajl.

### Preduslovi
* [Go 1.20+](https://go.dev/dl/)
* Git

### Pokretanje projekta

1. **Kloniranje repozitorijuma:**
   ```bash
   git clone https://github.com/msc-privacy-grid-mpc-zkp/smart-meter-simulator.git
   cd smart-meter-simulator

1. **Instalacija zavisnosti:**
   ```bash
   go mod tidy

3. **Pokretanje simulacije:**
   ```bash
   go run cmd/simulator/main.go

