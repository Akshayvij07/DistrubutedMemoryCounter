package di

import (
	"github.com/Akshayvij07/D-inmemory-counter/Internal/api"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/api/handler"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/cluster"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/counter"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/retryqueue"
)

func ConfigureServer(port string, peers []string) *api.Server {
	registry := node.NewServiceRegistry()
	retryQueue := retryqueue.NewRetryQueue()
	counter := counter.NewCounter()
	id := registry.RegisterNode(port)
	handler := handler.New(registry, counter, port, id, retryQueue)
	cluster := cluster.NewCluster(port, peers, registry, handler, retryQueue)
	cluster.StartNode()

	return api.NewServer(port, handler)
}
