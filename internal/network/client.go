package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

// Client represents an HTTP client optimized for high-throughput,
// concurrent network requests to the aggregator nodes.
type Client struct {
	ServerURL  string
	httpClient *http.Client
}

// NewClient initializes a new network client with a custom HTTP Transport.
// It configures connection pooling (MaxIdleConnsPerHost) to prevent TCP port
// exhaustion and reduce latency during parallel Zero-Knowledge Proof dispatch.
func NewClient(serverURL string) *Client {
	customTransport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
	}

	return &Client{
		ServerURL: serverURL,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: customTransport,
		},
	}
}

// SendProof serializes and transmits the cryptographic proof and MPC share
// to the designated aggregator node via a POST request.
func (c *Client) SendProof(payload ProofPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.ServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("network error while sending proof to server: %w", err)
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("[WARNING] Failed to close response body: %v\n", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server rejected the proof with status: %s", resp.Status)
	}

	return nil
}
