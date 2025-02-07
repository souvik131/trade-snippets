package queries

import (
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"gopkg.in/yaml.v2"
)

// QueryTemplate represents a single query template with its metadata
type QueryTemplate struct {
	Description string `yaml:"description"`
	Template    string `yaml:"template"`
}

// QueryConfig represents the structure of our YAML config file
type QueryConfig struct {
	Queries map[string]QueryTemplate `yaml:"queries"`
}

// Parameter represents a query parameter with its type
type Parameter struct {
	Name string
	Type string
}

// QueryService handles template loading and query execution
type QueryService struct {
	templates map[string]QueryTemplate
	conn      clickhouse.Conn
}

// NewQueryService creates a new QueryService instance
func NewQueryService(conn clickhouse.Conn) (*QueryService, error) {
	qs := &QueryService{
		templates: make(map[string]QueryTemplate),
		conn:      conn,
	}

	// Load all template files from the templates directory
	templatesDir := "queries/templates"
	files, err := ioutil.ReadDir(templatesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			content, err := ioutil.ReadFile(filepath.Join(templatesDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read template file %s: %v", file.Name(), err)
			}

			var config QueryConfig
			if err := yaml.Unmarshal(content, &config); err != nil {
				return nil, fmt.Errorf("failed to parse template file %s: %v", file.Name(), err)
			}

			// Add templates to the map
			for name, template := range config.Queries {
				qs.templates[name] = template
			}
		}
	}

	return qs, nil
}

// GetQueryParameters returns the required parameters for a query template
func (qs *QueryService) GetQueryParameters(templateName string) ([]Parameter, error) {
	template, ok := qs.templates[templateName]
	if !ok {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// Extract parameters using regex
	paramRegex := regexp.MustCompile(`\{([^:}]+):([^}]+)\}`)
	matches := paramRegex.FindAllStringSubmatch(template.Template, -1)

	params := make([]Parameter, 0, len(matches))
	for _, match := range matches {
		params = append(params, Parameter{
			Name: match[1],
			Type: match[2],
		})
	}

	return params, nil
}

// ListQueryTemplates returns all available query templates with their descriptions
func (qs *QueryService) ListQueryTemplates() map[string]string {
	templates := make(map[string]string)
	for name, template := range qs.templates {
		templates[name] = template.Description
	}
	return templates
}

// ExecuteQuery executes a query template with provided parameters and writes results to CSV
func (qs *QueryService) ExecuteQuery(ctx context.Context, templateName string, params map[string]interface{}, outputPath string) error {
	template, ok := qs.templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Replace parameters in the query
	query := template.Template
	for name, value := range params {
		var paramStr string
		switch v := value.(type) {
		case time.Time:
			paramStr = fmt.Sprintf("'%s'", v.Format("2006-01-02 15:04:05"))
		case string:
			paramStr = fmt.Sprintf("'%s'", v)
		default:
			paramStr = fmt.Sprintf("%v", v)
		}
		query = strings.Replace(query, fmt.Sprintf("{%s:%s}", name, getParamType(value)), paramStr, -1)
	}

	log.Println(query)

	// Execute query
	rows, err := qs.conn.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}
	defer rows.Close()

	// Create CSV file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	columnTypes := rows.ColumnTypes()
	headers := make([]string, len(columnTypes))
	for i, ct := range columnTypes {
		headers[i] = ct.Name()
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	// Get column names
	columns := rows.Columns()

	// Write data
	for rows.Next() {
		// Create a slice to hold the row values with appropriate types
		rowValues := make([]interface{}, len(columns))
		for i, ct := range columnTypes {
			switch ct.DatabaseTypeName() {
			case "String":
				rowValues[i] = new(string)
			case "Float64":
				rowValues[i] = new(float64)
			case "DateTime", "DateTime64":
				var t time.Time
				rowValues[i] = &t
			default:
				log.Printf("Column %s has type %s", ct.Name(), ct.DatabaseTypeName())
				rowValues[i] = new(string)
			}
		}

		// Scan the row into typed pointers
		if err := rows.Scan(rowValues...); err != nil {
			return fmt.Errorf("failed to scan row: %v", err)
		}

		// Convert values to strings
		record := make([]string, len(columns))
		for i, v := range rowValues {
			if v == nil {
				record[i] = ""
				continue
			}
			switch val := v.(type) {
			case *string:
				if val != nil {
					record[i] = *val
				}
			case *float64:
				if val != nil {
					record[i] = fmt.Sprintf("%v", *val)
				}
			case *time.Time:
				if val != nil {
					record[i] = val.Format(time.RFC3339)
				}
			default:
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %v", err)
		}
	}

	return rows.Err()
}

// Helper function to get parameter type string
func getParamType(value interface{}) string {
	switch value.(type) {
	case time.Time:
		return "DateTime"
	case string:
		return "String"
	case int, int32, int64:
		return "Int64"
	case float32, float64:
		return "Float64"
	default:
		return "String"
	}
}
