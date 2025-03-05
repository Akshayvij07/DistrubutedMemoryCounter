package helpers

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

func MustAtoi(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func RetryWithBackoff(fn func() error) error {
	maxRetries := 5
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		log.Printf("Retry %d/%d after error: %v", i+1, maxRetries, err)
		time.Sleep(backoff)
		backoff *= 2
	}

	log.Println("Max retries reached. Giving up.")
	return fmt.Errorf("max retries reached")
}

func IsPortAvailable(port string) bool {
	// Resolve the port to a UDP address
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		log.Printf("Failed to resolve port %s: %v\n", port, err)
		return false
	}

	// Attempt to listen on the port
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Printf("Port %s is not available: %v\n", port, err)
		return false
	}
	defer conn.Close()

	log.Printf("Port %s is available\n", port)
	return true
}
