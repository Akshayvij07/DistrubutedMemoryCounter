package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

func ListenForOtherNodes(id string, cluster *ServiceRegistry) {
	for {
		time.Sleep(5 * time.Second) // Gossip every 5 seconds
		peer, _ := cluster.GetRandomPeer(id)
		log.Println("Gossiping with", peer)
		if peer != "" {
			go gossipPeerList(peer, cluster)
		}
	}
}

func gossipPeerList(peer string, cluster *ServiceRegistry) []Node {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		log.Printf("Failed to connect to peer: %s\n", peer)
		return nil
	}
	defer conn.Close()

	fmt.Fprintln(conn, "GOSSIP")
	decoder := json.NewDecoder(conn)
	var newPeers []Node
	if err := decoder.Decode(&newPeers); err == nil {
		for _, node := range newPeers {
			// Prevent duplicate registrations
			if node.ID != "" {
				cluster.RegisterNode(node.ID)
			}
		}
	}
	return newPeers
}
