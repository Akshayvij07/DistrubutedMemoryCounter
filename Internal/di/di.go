package di

import (
	"github.com/Akshayvij07/D-inmemory-counter/Internal/api"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/api/handler"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/counter"
	"github.com/Akshayvij07/D-inmemory-counter/Internal/services/node"
)

func ConfigureServer(port string, peers []string) *api.Server {
	registry := node.NewServiceRegistry()
	counter := counter.NewCounter()
	id := registry.RegisterNode(port)
	handler := handler.New(registry, counter, port, id)
	node.StartNode(port, peers, registry)

	return api.NewServer(port, handler)
}
