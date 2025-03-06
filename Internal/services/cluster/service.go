package cluster

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/api/handler"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/retryqueue"
	"github.com/Akshayvij07/D-inmemory-counter/pkg/helpers"
)

type Cluster struct {
	handler         *handler.Handler
	peers           []string
	port            string
	ServiceRegistry *node.ServiceRegistry
	retryQueue      *retryqueue.RetryQueue
}

func NewCluster(port string, peers []string, registry *node.ServiceRegistry, handler *handler.Handler, retryQueue *retryqueue.RetryQueue) *Cluster {
	return &Cluster{
		handler:         handler,
		peers:           peers,
		port:            port,
		ServiceRegistry: registry,
		retryQueue:      retryQueue,
	}
}

func (c *Cluster) StartNode() {

	// c.ServiceRegistry.RegisterNode(c.port)
	if len(c.peers) > 0 {
		var allPeers []node.Node
		for _, peer := range c.peers {
			peerList := c.gossipPeerList(peer)
			if len(peerList) > 0 {
				allPeers = append(allPeers, peerList...)
			}
		}
		// Merge the received peer lists into the registry
		c.ServiceRegistry.MergeNodes(allPeers)
	}

	go func() {
		for {
			peers := c.ServiceRegistry.ListNodes()
			for _, peer := range peers {
				if peer.Unresponsive {
					if helpers.IsPortAvailable(peer.Port) {
						c.ServiceRegistry.MarkPeerResponsive(peer.ID)
					}
				}
			}
			time.Sleep(5 * time.Second) // Adjust retry interval as needed
		}
	}()

	go c.monitorUnresponsivePeers()

	// Start a goroutine to periodically retry failed requests
	go c.retryFailedRequests()

	// go func() {
	// 	for {
	// 		c.handler.RetryRequest()
	// 		time.Sleep(5 * time.Second) // Adjust retry interval as needed
	// 	}
	// }()

	go c.sendHeartbeats()
	go c.broadcastNewNode()
	go c.listenForHeartbeats()
	go c.RemoveUnresponsiveNodes()
	go c.ListenForOtherNodes()
}

func (c *Cluster) monitorUnresponsivePeers() {
	for {
		peers := c.ServiceRegistry.ListNodes()
		for _, peer := range peers {
			if peer.Unresponsive && helpers.IsPortAvailable(peer.Port) {

				log.Println("Marking peer", peer.ID, "as responsive")

				c.ServiceRegistry.MarkPeerResponsive(peer.ID)
			}
		}
		time.Sleep(5 * time.Second) // Adjust interval as needed
	}
}

// retryFailedRequests periodically processes the retry queue
func (c *Cluster) retryFailedRequests() {
	for {
		c.handler.RetryRequest()
		time.Sleep(5 * time.Second) // Adjust interval as needed
	}
}

func (c *Cluster) broadcastNewNode() {
	for _, peer := range c.peers {
		go func(peer string) {
			conn, err := net.DialTimeout("tcp", peer, 3*time.Second)
			if err != nil {
				log.Printf("Failed to broadcast to peer %s: %v\n", peer, err)
				return
			}
			defer conn.Close()

			// Send the new node's address
			if _, err := fmt.Fprintln(conn, c.port); err != nil {
				log.Printf("Failed to send new node info to %s: %v\n", peer, err)
			}
		}(peer)
	}
}
func (c *Cluster) sendHeartbeats() {
	for {
		time.Sleep(5 * time.Second) // Send every 5 seconds
		for _, peer := range c.peers {
			addr, _ := net.ResolveUDPAddr("udp", peer)
			conn, err := net.DialUDP("udp", nil, addr)
			if err == nil {
				conn.Write([]byte(c.port))
				conn.Close()
			}
		}
		c.ServiceRegistry.UpdateLastSeen(c.port)
	}
}

// Clean up nodes that missed heartbeats

func (c *Cluster) RemoveUnresponsiveNodes() {
	waitTime := 60 * time.Second
	for {
		time.Sleep(waitTime) // Check every 60 seconds
		c.ServiceRegistry.RemoveUnresponsiveNodes(waitTime)
	}
}
func (c *Cluster) listenForHeartbeats() bool {
	p, err := net.LookupPort("udp", c.port)
	if err != nil {
		log.Fatalf("Invalid port: %s", c.port)
	}
	addr := net.UDPAddr{Port: p}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Failed to listen for heartbeats on port %s: %v", c.port, err)
	}
	defer conn.Close()

	log.Printf("Listening for heartbeats on port %s\n", c.port)
	buf := make([]byte, 1024)
	for {
		n, remoteAddr, _ := conn.ReadFromUDP(buf)
		peerID := string(buf[:n])
		c.ServiceRegistry.RegisterNode(peerID)
		log.Printf("Received heartbeat from %s\n", remoteAddr.String())
	}
}

func (c *Cluster) ListenForOtherNodes() {
	for {
		time.Sleep(5 * time.Second) // Gossip every 5 seconds
		peer, _ := c.ServiceRegistry.GetRandomPeer(c.port)
		log.Println("Gossiping with", peer)
		if peer != "" {
			go c.gossipPeerList(peer)
		}
	}
}

func (c *Cluster) gossipPeerList(peer string) []node.Node {
	conn, err := net.Dial("tcp", peer)
	if err != nil {
		log.Printf("Failed to connect to peer: %s\n", peer)
		return nil
	}
	defer conn.Close()

	fmt.Fprintln(conn, "GOSSIP")
	decoder := json.NewDecoder(conn)
	var newPeers []node.Node
	if err := decoder.Decode(&newPeers); err == nil {
		for _, node := range newPeers {
			// Prevent duplicate registrations
			if node.ID != "" {
				c.ServiceRegistry.RegisterNode(node.ID)
			}
		}
	}
	return newPeers
}
