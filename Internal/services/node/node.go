package node

import (
	"log"
	"sort"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type Registry interface {
	RegisterNode(port string) string
	ListNodes() []Node
	RemoveNode(id string)
	GetRandomPeer(exclude string) string
}

// Node represents a service instance
type Node struct {
	ID       string
	Address  string
	LastSeen time.Time
}

// ServiceRegistry stores active nodes
type ServiceRegistry struct {
	mu    sync.Mutex
	nodes map[string]*Node
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		nodes: make(map[string]*Node),
	}
}

// RegisterNode adds a node to the registry
func (r *ServiceRegistry) RegisterNode(port string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := "localhost:" + port

	node, exists := r.nodes[id] // Get the node if it exists
	if !exists {
		node = &Node{ID: id, Address: port, LastSeen: time.Now()}
		log.Printf("Node registered: %s\n", id)
		r.nodes[id] = node // Add to registry only when it's a new node
	} else {
		log.Printf("Node already exists, updating LastSeen: %s\n", id)
	}
	node.LastSeen = time.Now() // Update LastSeen
	return id
}

// RemoveNode removes a node from the registry
func (r *ServiceRegistry) RemoveNode(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.nodes, id)
	log.Printf("Node removed: %s\n", id)
}

// ListNodes returns active nodes
func (r *ServiceRegistry) ListNodes() []Node {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := []Node{}
	for _, node := range r.nodes {
		nodes = append(nodes, *node)
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	return nodes
}
func (r *ServiceRegistry) ListNodesAddress() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := []string{}
	for _, node := range r.nodes {
		nodes = append(nodes, node.Address)
	}

	return nodes
}

func (r *ServiceRegistry) MergeNodes(newNodes []Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, node := range newNodes {
		if _, exists := r.nodes[node.Address]; !exists {
			r.nodes[node.Address] = &Node{
				ID:       node.Address,
				Address:  node.Address,
				LastSeen: time.Now(),
			}
			log.Printf("Node %s added to registry\n", node.Address)
		}
	}
}

func (c *ServiceRegistry) GetRandomPeer(exclude string) string {
	c.mu.Lock()
	defer c.mu.Unlock()

	peers := make([]string, 0, len(c.nodes))
	for id := range c.nodes {
		if id != exclude {
			peers = append(peers, id)
		}
	}

	if len(peers) == 0 {
		return ""
	}

	return peers[rand.Intn(len(peers))] // Select a random peer
}
