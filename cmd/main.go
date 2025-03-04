package main

import (
	"flag"
	"log"
	"strings"

	"github.com/Akshayvij07/D-inmemory-counter/Internal/di"
)

func main() {
	port := flag.String("port", "", "Port to run the node on")
	peers := flag.String("peers", "", "Comma-separated list of peer addresses")
	flag.Parse()

	// Validate port
	if *port == "" {
		log.Fatal("Port is required. Use -port flag to specify the port.")
	}

	// Convert peers string to a slice
	peerList := []string{}
	if *peers != "" {
		peerList = strings.Split(*peers, ",")
	}

	server := di.ConfigureServer(*port, peerList)
	// Start the server
	if err := server.Serve(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("Usage: go run main.go <port>")
// 		return
// 	}
// 	port := os.Args[1]

// 	// Define peers directly in the code
// 	peerMap := map[string][]string{
// 		"8080": {"8081", "8082"},
// 		"8081": {"8080", "8082"},
// 		"8082": {"8080", "8081"},
// 	}

// 	peers, exists := peerMap[port]
// 	if !exists {
// 		log.Printf("No predefined peers for port %s\n", port)
// 		peers = []string{}
// 	}

// 	// Format peers with localhost
// 	for i := range peers {
// 		peers[i] = "localhost:" + peers[i]
// 	}

// 	log.Println("Starting node on port:", port)
// 	log.Println("Peers:", peers)

// 	cluster := node.NewServiceRegistry()
// 	node.StartNode(port, peers, cluster)

// 	select {} // Keep running
// }
