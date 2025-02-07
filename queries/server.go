package queries

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/souvik131/trade-snippets/analytics"
)

// StartServer initializes and starts the HTTP server for query API
func StartServer(port int) error {
	// Create query service using existing ClickHouse connection
	queryService, err := NewQueryService(analytics.GetConnection())
	if err != nil {
		return fmt.Errorf("failed to create query service: %v", err)
	}

	// Create query handler
	queryHandler := NewQueryHandler(queryService)

	// Setup gin router
	router := gin.Default()

	// Register query routes
	queryHandler.RegisterRoutes(router)

	// Start server
	addr := fmt.Sprintf(":%d", port)
	if err := router.Run(addr); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	return nil
}
