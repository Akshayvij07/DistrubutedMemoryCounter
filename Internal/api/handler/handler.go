package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/counter"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/retryqueue"
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
	if requestID != "" {
		n.counter.Increment(requestID)
		c.JSON(http.StatusOK, gin.H{"message": "Incremented"})
		return
	}
	cRequestID := c.GetHeader("Client-Request-ID")
	if cRequestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing X-Request-ID header"})
		return
	}

	if n.counter.WasProcessed(cRequestID) {
		log.Printf("Skipping already processed request ID: %s", cRequestID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate request ID"})
		return
	}

	n.counter.Increment(cRequestID)
	n.propagateIncrement(cRequestID)
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
	req.Header.Set("X-Origin-Node", n.id) // Include the origin node ID

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
	log.Printf("RetryRequest called\n")

	requests := n.retryQueue.GetRetryRequest()

	log.Println("Number of requests to retry:", len(requests))

	if len(requests) == 0 {
		return // No requests to retry
	}

	for _, request := range requests {
		go func(req retryqueue.RetryRequest) {
			limiter := time.NewTicker(100 * time.Millisecond)
			defer limiter.Stop()
			n.processRetryRequest(req, limiter)
		}(request)
	}
}

// processRetryRequest handles a single retry request with exponential backoff
func (n *Handler) processRetryRequest(request retryqueue.RetryRequest, limiter *time.Ticker) {
	log.Print("processRetryRequest called\n")
	// Check if the peer is unresponsive
	if n.registry.IsPeerUnresponsive(request.Peer.Port) {
		log.Printf("Peer %s is unresponsive. Skipping retry for now.\n", request.Peer.ID)
		n.retryQueue.Enqueue(request.Peer, request.RequestID) // Re-enqueue for later retry
		return
	}

	log.Print("Peer is responsive\n")

	backoff := 1 * time.Second
	maxRetries := 5
	maxBackoff := 30 * time.Second

	for i := 0; i < maxRetries; i++ {
		<-limiter.C

		err := n.SendIncrementRequest(request.Peer, request.RequestID)
		if err == nil {
			log.Printf("Successfully retried increment to peer %s\n", request.Peer.ID)
			n.registry.MarkPeerResponsive(request.Peer.ID)
			n.retryQueue.Dequeue()
			return
		}

		log.Printf("Failed to retry increment to peer %s (attempt %d/%d): %v\n", request.Peer.ID, i+1, maxRetries, err)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	time.AfterFunc(5*time.Second, func() {
		n.retryQueue.Enqueue(request.Peer, request.RequestID)
	})
	log.Printf("Max retries reached for peer %s. Re-enqueuing request.\n", request.Peer.ID)
}
