package node

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"golang.org/x/exp/rand"
)

type Registry interface {
	MarkPeerResponsive(peer string)
	IsPeerUnresponsive(id string) bool
	MarkPeerUnresponsive(peer string)
	RegisterNode(address string) string
	ListNodes() []Node
	RemoveNode(id string)
	GetRandomPeer(exclude string) (string, bool)
	MergeNodes(newNodes []Node)
	Cleanup(timeout time.Duration)
	Size() int
	// AppendCounter(peer string)
}

// Node represents a service instance
type Node struct {
	ID           string
	Address      string
	Port         string
	Unresponsive bool
	LastSeen     time.Time
}

func (n Node) String() string {
	return fmt.Sprintf("Node{ID: %s, Address: %s, LastSeen: %s}", n.ID, n.Address, n.LastSeen.Format(time.RFC3339))
}

// ServiceRegistry stores active nodes
type ServiceRegistry struct {
	mu    sync.Mutex
	nodes map[string]*Node
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		mu:    sync.Mutex{},
		nodes: make(map[string]*Node),
	}
}

// RegisterNode adds a node to the registry
func (r *ServiceRegistry) RegisterNode(port string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	address := "localhost:" + port

	node, exists := r.nodes[address]
	if !exists {
		node = &Node{ID: address, Address: address, Port: port, LastSeen: time.Now()}
		log.Printf("Node registered: %s\n", address)
		r.nodes[address] = node
	} else {
		log.Printf("Node already exists, updating LastSeen: %s\n", address)
	}
	node.LastSeen = time.Now()
	return address
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

// ListNodesAddress returns unique node addresses
func (r *ServiceRegistry) ListNodesAddress() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	uniqueAddresses := make(map[string]struct{})
	for _, node := range r.nodes {
		uniqueAddresses[node.Address] = struct{}{}
	}

	addresses := make([]string, 0, len(uniqueAddresses))
	for addr := range uniqueAddresses {
		addresses = append(addresses, addr)
	}

	return addresses
}

// MergeNodes merges a list of nodes into the registry
func (r *ServiceRegistry) MergeNodes(newNodes []Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, node := range newNodes {
		if _, exists := r.nodes[node.ID]; !exists {
			r.nodes[node.ID] = &node
			log.Printf("Node %s added to registry\n", node.ID)
		}
	}
}

// GetRandomPeer returns a random peer, excluding the specified node
func (c *ServiceRegistry) GetRandomPeer(exclude string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	peers := make([]string, 0, len(c.nodes))
	for id := range c.nodes {
		if id != exclude {
			peers = append(peers, id)
		}
	}

	if len(peers) == 0 {
		return "", false
	}

	return peers[rand.Intn(len(peers))], true
}

// Cleanup removes nodes that haven't been seen within the timeout duration
func (r *ServiceRegistry) Cleanup(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, node := range r.nodes {
		if time.Since(node.LastSeen) > timeout {
			delete(r.nodes, id)
			log.Printf("Node %s removed due to timeout\n", id)
		}
	}
}

// Size returns the number of active nodes
func (r *ServiceRegistry) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.nodes)
}

func (n *ServiceRegistry) MarkPeerUnresponsive(peer string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, p := range n.nodes {
		if p.Address == peer {
			n.nodes[i].Unresponsive = true
			n.nodes[i].LastSeen = time.Now()
			break
		}
	}
}

func (n *ServiceRegistry) MarkPeerResponsive(peer string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	for i, p := range n.nodes {
		if p.Address == peer {
			n.nodes[i].Unresponsive = false
			n.nodes[i].LastSeen = time.Now()
			break
		}
	}
}

func (sr *ServiceRegistry) UpdateLastSeen(nodeID string) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if node, exists := sr.nodes[nodeID]; exists {
		node.LastSeen = time.Now()
	}
}

func (sr *ServiceRegistry) RemoveUnresponsiveNodes(timeout time.Duration) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	for id, node := range sr.nodes {
		if time.Since(node.LastSeen) > timeout {
			delete(sr.nodes, id)
			log.Printf("Node %s removed due to timeout\n", id)
		}
	}
}

func (sr *ServiceRegistry) IsPeerUnresponsive(id string) bool {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if node, exists := sr.nodes[id]; exists {
		return node.Unresponsive
	}
	return false
}

// func (sr *ServiceRegistry) AppendCounter(peer string) {
// 	sr.mu.Lock()
// 	defer sr.mu.Unlock()

// 	if node, exists := sr.nodes[peer]; exists {
// 		if _, ok := node.Counters[peer]; ok {
// 			node.Counters[peer] += 1
// 		}
// 	}

// }
