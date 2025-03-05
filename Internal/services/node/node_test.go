package node

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestServiceRegistry(t *testing.T) {
	registry := NewServiceRegistry()

	// Test RegisterNode
	id1 := registry.RegisterNode("8080")
	id2 := registry.RegisterNode("9090")
	if id1 != "localhost:8080" || id2 != "localhost:9090" {
		t.Errorf("RegisterNode failed: got %s and %s", id1, id2)
	}

	// Test ListNodes
	nodes := registry.ListNodes()
	if len(nodes) != 2 {
		t.Errorf("ListNodes failed: expected 2 nodes, got %d", len(nodes))
	}

	// Test ListNodesAddress
	nodeAddresses := registry.ListNodesAddress()
	if len(nodeAddresses) != 2 || nodeAddresses[0] != "8080" || nodeAddresses[1] != "9090" {
		t.Errorf("ListNodesAddress failed: got %v", nodeAddresses)
	}

	// Test RemoveNode
	registry.RemoveNode("localhost:8080")
	nodesAfterRemoval := registry.ListNodes()
	if len(nodesAfterRemoval) != 1 {
		t.Errorf("RemoveNode failed: expected 1 node, got %d", len(nodesAfterRemoval))
	}

	// Test MergeNodes
	newNodes := []Node{{ID: "localhost:7070", Address: "7070", LastSeen: time.Now()}}
	registry.MergeNodes(newNodes)
	nodesAfterMerge := registry.ListNodes()
	if len(nodesAfterMerge) != 2 {
		t.Errorf("MergeNodes failed: expected 2 nodes, got %d", len(nodesAfterMerge))
	}

	// Test GetRandomPeer
	exclude := "localhost:7070"
	randomPeer, _ := registry.GetRandomPeer(exclude)
	if randomPeer == exclude {
		t.Errorf("GetRandomPeer failed: returned excluded peer %s", randomPeer)
	}
}

func TestConcurrentRegisterNode(t *testing.T) {
	registry := NewServiceRegistry()

	var wg sync.WaitGroup
	numNodes := 100
	ports := make([]string, numNodes)

	// Generate 100 unique ports for testing
	for i := 0; i < numNodes; i++ {
		ports[i] = "80" + strconv.Itoa(i)
	}

	// Concurrently register nodes
	for _, port := range ports {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			registry.RegisterNode(p)
		}(port)
	}

	wg.Wait()

	// Verify that all nodes are registered
	nodes := registry.ListNodes()
	if len(nodes) != numNodes {
		t.Errorf("Expected %d nodes, but got %d", numNodes, len(nodes))
	}
}
