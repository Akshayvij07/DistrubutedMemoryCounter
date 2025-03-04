package node

import (
	"fmt"
	"log"
	"net"
	"time"
)

func StartNode(port string, peers []string, ServiceRegistry *ServiceRegistry) {
	ServiceRegistry.RegisterNode(port)
	if len(peers) > 0 {
		var allPeers []Node
		for _, peer := range peers {
			peerList := gossipPeerList(peer, ServiceRegistry)
			if len(peerList) > 0 {
				allPeers = append(allPeers, peerList...)
			}
		}
		// Merge the received peer lists into the registry
		ServiceRegistry.MergeNodes(allPeers)
	}

	go sendHeartbeats(port, peers, ServiceRegistry)
	go broadcastNewNode(port, peers)
	go listenForHeartbeats(port, ServiceRegistry)
	go RemoveUnresponsiveNodes(ServiceRegistry)
	go ListenForOtherNodes(port, ServiceRegistry)
}

func broadcastNewNode(newNode string, peers []string) {
	for _, peer := range peers {
		go func(peer string) {
			conn, err := net.DialTimeout("tcp", peer, 3*time.Second)
			if err != nil {
				log.Printf("Failed to broadcast to peer %s: %v\n", peer, err)
				return
			}
			defer conn.Close()

			// Send the new node's address
			if _, err := fmt.Fprintln(conn, newNode); err != nil {
				log.Printf("Failed to send new node info to %s: %v\n", peer, err)
			}
		}(peer)
	}
}
func sendHeartbeats(id string, peers []string, registry *ServiceRegistry) {
	for {
		time.Sleep(5 * time.Second) // Send every 5 seconds
		for _, peer := range peers {
			addr, _ := net.ResolveUDPAddr("udp", peer)
			conn, err := net.DialUDP("udp", nil, addr)
			if err == nil {
				conn.Write([]byte(id))
				conn.Close()
			}
		}
		registry.mu.Lock()
		if node, exists := registry.nodes[id]; exists {
			node.LastSeen = time.Now()
		}
		registry.mu.Unlock()
	}
}

// Clean up nodes that missed heartbeats
func RemoveDeadNodes(registry *ServiceRegistry) {
	for {
		time.Sleep(60 * time.Second) // Check every 30 seconds
		registry.mu.Lock()
		for id, node := range registry.nodes {
			if time.Since(node.LastSeen) > 30*time.Second { // Remove after 30 seconds
				delete(registry.nodes, id)
				log.Printf("Node %s removed due to timeout\n", id)
			}
		}
		registry.mu.Unlock()
	}
}

func RemoveUnresponsiveNodes(registry *ServiceRegistry) {
	for {
		time.Sleep(60 * time.Second) // Check every 60 seconds
		registry.mu.Lock()
		for id, node := range registry.nodes {
			if time.Since(node.LastSeen) > 60*time.Second { // Remove after 60 seconds
				delete(registry.nodes, id)
				log.Printf("Node %s removed due to timeout\n", id)
			}
		}
		registry.mu.Unlock()
	}
}
func listenForHeartbeats(port string, registry *ServiceRegistry) {
	p, err := net.LookupPort("udp", port)
	if err != nil {
		log.Fatalf("Invalid port: %s", port)
	}
	addr := net.UDPAddr{Port: p}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to listen for heartbeats on port %s: %v", port, err)
	}
	defer conn.Close()

	log.Printf("Listening for heartbeats on port %s\n", port)
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, _ := conn.ReadFromUDP(buf)
		peerID := string(buf[:n])
		registry.RegisterNode(peerID)
		log.Printf("Received heartbeat from %s\n", remoteAddr.String())
	}
}
