package api

import (
	"github.com/Akshayvij07/D-inmemory-counter/Internal/api/handler"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Server struct {
	router *gin.Engine
	Port   string
}

func (s *Server) Serve() error {
	return s.router.Run("0.0.0.0:" + s.Port)
}

// NewServer initializes a new server
func NewServer(port string, handler *handler.Handler) *Server {
	router := gin.New()
	gin.SetMode(gin.ReleaseMode) // Use gin.ReleaseMode for production

	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Define routes
	router.POST("/increment", handler.IncrementHandler)
	router.GET("/count", handler.CountHandler)

	log.Printf("Server started on port %v\n", port)

	return &Server{
		router: router,
		Port:   port,
	}
}
