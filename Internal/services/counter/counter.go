package counter

import (
	"log"
	"sync"
)

type CounterService interface {
	Increment(requestID string)
	WasProcessed(requestID string) bool
	GetCount() int
	AppendCounter(peerID string)
}

type Counter struct {
	mu         sync.Mutex
	count      int
	seenIDs    map[string]bool
	peersCount map[string]int
}

func NewCounter() *Counter {
	return &Counter{
		seenIDs: make(map[string]bool),
	}
}

func (c *Counter) Increment(requestID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seenIDs[requestID] {
		log.Printf("Request %s already processed. Skipping.", requestID)
		return
	}
	c.count++
	c.seenIDs[requestID] = true
}

func (c *Counter) GetCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

func (c *Counter) WasProcessed(requestID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.seenIDs[requestID]
}

func (c *Counter) AppendCounter(peerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.peersCount[peerID]; ok {
		c.peersCount[peerID] += 1
	}
}

func (c *Counter) GetPeersCount() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.peersCount
}
