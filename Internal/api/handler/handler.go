package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/counter"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/retryqueue"
	"github.com/Akshayvij07/D-inmemory-counter/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	counter    counter.CounterService
	registry   node.Registry
	client     *http.Client
	id         string
	retryQueue *retryqueue.RetryQueue
}

func New(registry node.Registry, counter counter.CounterService, port string, id string, retryQueue *retryqueue.RetryQueue) *Handler {
	// id := registry.RegisterNode(port)
	return &Handler{
		counter:    counter,
		registry:   registry,
		client:     &http.Client{Timeout: 10 * time.Second},
		id:         id,
		retryQueue: retryQueue,
	}
}
func (n *Handler) IncrementHandler(c *gin.Context) {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Request-ID header"})
		return
	}

	if n.counter.WasProcessed(requestID) {
		log.Printf("Skipping already processed request ID: %s", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate request ID"})
		return
	}

	n.counter.Increment(requestID)
	n.propagateIncrement(requestID)
	c.JSON(http.StatusOK, gin.H{"message": "Incremented"})
}

// CountHandler handles the /count endpoint
func (n *Handler) CountHandler(c *gin.Context) {
	count := n.counter.GetCount()
	c.JSON(http.StatusOK, gin.H{"count": count})
}

func (n *Handler) getPeersExceptSelf() []node.Node {
	peers := n.registry.ListNodes()
	var peersExceptSelf []node.Node
	for _, peer := range peers {
		if peer.ID != n.id {
			peersExceptSelf = append(peersExceptSelf, peer)
		}
	}
	return peersExceptSelf
}

func (n *Handler) propagateIncrement(requestID string) {

	peers := n.getPeersExceptSelf()

	log.Printf("Propagating increment to %d peers", len(peers))

	log.Println("Peers:", peers)

	for _, peer := range peers {
		go func(peer node.Node) {
			if err := n.SendIncrementRequest(peer, requestID); err != nil {
				log.Println("Error sending increment:", err)
			}
		}(peer)
	}
}

func (n *Handler) SendIncrementRequest(peer node.Node, requestID string) error {
	url := fmt.Sprintf("http://%s/increment", peer.ID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Request-ID", requestID)

	log.Printf("Sending increment to %s with request ID: %s", peer.ID, requestID)

	resp, err := n.client.Do(req)
	if err != nil {
		// Enqueue the failed request
		n.registry.MarkPeerUnresponsive(peer.ID)
		n.retryQueue.Enqueue(peer, requestID)
		return err
	}
	defer resp.Body.Close()

	log.Printf("Response from %s: %s", peer.ID, resp.Status)

	if resp.StatusCode != http.StatusOK {
		// Enqueue the failed request
		n.retryQueue.Enqueue(peer, requestID)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	n.counter.AppendCounter(peer.ID)

	return nil
}
func (n *Handler) RetryRequest() {
	requests := n.retryQueue.GetRetryRequest()
	if len(requests) > 0 {
		limiter := time.NewTicker(100 * time.Millisecond) // Allow 10 retries per second
		defer limiter.Stop()

		for _, request := range requests {
			go func(request retryqueue.RetryRequest) {
				// Check if the peer is unresponsive
				if n.registry.IsPeerUnresponsive(request.Peer.Port) {
					log.Printf("Peer %s is unresponsive. Skipping retry for now.\n", request.Peer.ID)
					if helpers.IsPortAvailable(request.Peer.Port) {
						n.registry.MarkPeerResponsive(request.Peer.Port)
					}
					n.retryQueue.Enqueue(request.Peer, request.RequestID) // Re-enqueue for later retry
					return
				}

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // Set a timeout
				defer cancel()

				// Exponential backoff
				backoff := 1 * time.Second
				maxRetries := 5

				for i := 0; i < maxRetries; i++ {
					select {
					case <-ctx.Done():
						log.Printf("Retry for peer %s canceled: %v\n", request.Peer.ID, ctx.Err())
						return
					case <-limiter.C: // Rate limit
						err := n.SendIncrementRequest(request.Peer, request.RequestID)
						if err == nil {
							log.Printf("Successfully retried increment to peer %s\n", request.Peer.ID)
							return
						}

						log.Printf("Failed to retry increment to peer %s (attempt %d/%d): %v\n", request.Peer.ID, i+1, maxRetries, err)
						time.Sleep(backoff)
						backoff *= 2 // Double the backoff time
					}
				}

				// If all retries fail, re-enqueue the request
				n.retryQueue.Enqueue(request.Peer, request.RequestID)
				log.Printf("Max retries reached for peer %s. Re-enqueuing request.\n", request.Peer.ID)
			}(request)
		}
	}
}
