package queries

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	service *QueryService
}

func NewQueryHandler(service *QueryService) *QueryHandler {
	return &QueryHandler{
		service: service,
	}
}

// RegisterRoutes registers the query endpoints with the gin router
func (h *QueryHandler) RegisterRoutes(router *gin.Engine) {
	queryGroup := router.Group("/api/queries")
	{
		queryGroup.GET("/templates", h.listTemplates)
		queryGroup.GET("/templates/:name/parameters", h.getParameters)
		queryGroup.POST("/execute/:name", h.executeQuery)
		queryGroup.GET("/download/:filename", h.downloadCSV)
	}
}

// listTemplates returns all available query templates
func (h *QueryHandler) listTemplates(c *gin.Context) {
	templates := h.service.ListQueryTemplates()
	c.JSON(http.StatusOK, templates)
}

// getParameters returns the required parameters for a query template
func (h *QueryHandler) getParameters(c *gin.Context) {
	name := c.Param("name")
	params, err := h.service.GetQueryParameters(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, params)
}

type ExecuteQueryRequest struct {
	Parameters map[string]interface{} `json:"parameters"`
}

// executeQuery executes a query template and returns the results as CSV
func (h *QueryHandler) executeQuery(c *gin.Context) {
	name := c.Param("name")

	var req ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Process parameters
	params := make(map[string]interface{})
	for k, v := range req.Parameters {
		// Handle datetime parameters
		if str, ok := v.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				params[k] = t
				continue
			}
		}
		params[k] = v
	}

	// Create output directory if it doesn't exist
	outputDir := "csv"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output directory"})
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s_%d.csv", name, time.Now().Unix())
	outputPath := filepath.Join(outputDir, filename)

	// Execute query
	if err := h.service.ExecuteQuery(c.Request.Context(), name, params, outputPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Query executed successfully",
		"file":    filename,
	})
}

// downloadCSV serves the CSV file for download
func (h *QueryHandler) downloadCSV(c *gin.Context) {
	filename := c.Param("filename")
	filepath := filepath.Join("csv", filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv")
	c.File(filepath)
}
