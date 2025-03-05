package retryqueue

import (
	"log"
	"sync"
	"time"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
)

type RetryQueue struct {
	mu    sync.Mutex
	queue []RetryRequest
}

type RetryRequest struct {
	Peer      node.Node
	RequestID string
}

// NewRetryQueue creates a new RetryQueue instance.
func NewRetryQueue() *RetryQueue {
	return &RetryQueue{
		queue: make([]RetryRequest, 0),
	}
}

// Enqueue adds a failed request to the retry queue.
func (rq *RetryQueue) Enqueue(peer node.Node, requestID string) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.queue = append(rq.queue, RetryRequest{Peer: peer, RequestID: requestID})
}

// Dequeue removes and returns the first request from the retry queue.
func (rq *RetryQueue) Dequeue() (RetryRequest, bool) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if len(rq.queue) == 0 {
		return RetryRequest{}, false
	}

	request := rq.queue[0]
	rq.queue = rq.queue[1:]
	return request, true
}

// StartRetryWorker starts a goroutine to periodically retry failed requests.
func (rq *RetryQueue) StartRetryWorker(retryFunc func(peer node.Node, requestID string) error) {
	go func() {
		for {
			time.Sleep(5 * time.Second) // Adjust retry interval as needed

			for {
				request, ok := rq.Dequeue()
				if !ok {
					break
				}

				err := retryFunc(request.Peer, request.RequestID)
				if err != nil {
					// Re-enqueue if the retry fails
					rq.Enqueue(request.Peer, request.RequestID)
					log.Printf("Failed to retry increment to peer %s: %v\n", request.Peer.ID, err)
				} else {
					log.Printf("Successfully retried increment to peer %s\n", request.Peer.ID)
				}
			}
		}
	}()
}

func (rq *RetryQueue) GetRetryRequest() []RetryRequest {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	// Copy the queue and clear it
	requests := make([]RetryRequest, len(rq.queue))
	copy(requests, rq.queue)
	rq.queue = rq.queue[:0] // Clear the queue

	return requests
}
