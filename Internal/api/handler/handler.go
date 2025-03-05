package handler

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/counter"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
	"github.com/Akshayvij07/D-inmemory-counter/pkg/helpers"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	counter  counter.CounterService
	registry node.Registry
	client   *http.Client
	id       string
}

func New(registry node.Registry, counter counter.CounterService, port string, id string) *Handler {
	// id := registry.RegisterNode(port)
	return &Handler{
		counter:  counter,
		registry: registry,
		client:   &http.Client{Timeout: 10 * time.Second},
		id:       id,
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

func (n *Handler) sendIncrementRequest(peer node.Node, requestID string) error {
	url := fmt.Sprintf("http://%s/increment", peer.ID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Request-ID", requestID)

	log.Printf("Sending increment to %s with request ID: %s", peer.ID, requestID)

	return helpers.RetryWithBackoff(func() error {
		resp, err := n.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		log.Printf("Response from %s: %s", peer.ID, resp.Status)

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return nil
	})
}

func (n *Handler) propagateIncrement(requestID string) {

	peers := n.getPeersExceptSelf()

	log.Printf("Propagating increment to %d peers", len(peers))

	log.Println("Peers:", peers)

	for _, peer := range peers {
		go func(peer node.Node) {
			if err := n.sendIncrementRequest(peer, requestID); err != nil {
				log.Println("Error sending increment:", err)
			}
		}(peer)
	}
}
