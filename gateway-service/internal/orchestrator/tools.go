package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// ToolInput represents the structured input passed to a tool
type ToolInput struct {
	Args map[string]interface{} `json:"args"`
}

// ToolOutput represents the structured result of tool execution
type ToolOutput struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

// ToolDefinition describes a tool available to the agent
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
	Handler     ToolHandler            `json:"-"` // Unmarshallable function ref
}

// ToolHandler is the function signature for executing tools
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// ToolRegistry manages available tools and their execution
type ToolRegistry struct {
	tools map[string]*ToolDefinition
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDefinition),
	}
}

// Register adds a new tool to the registry
func (tr *ToolRegistry) Register(def *ToolDefinition) error {
	if def.Name == "" || def.Handler == nil {
		return fmt.Errorf("tool name and handler required")
	}
	tr.tools[def.Name] = def
	log.Printf("📦 Tool registered: %s", def.Name)
	return nil
}

// Get retrieves a tool definition by name
func (tr *ToolRegistry) Get(name string) *ToolDefinition {
	return tr.tools[name]
}

// List returns all registered tool names and descriptions
func (tr *ToolRegistry) List() []map[string]interface{} {
	var result []map[string]interface{}
	for _, tool := range tr.tools {
		result = append(result, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"schema":      tool.InputSchema,
		})
	}
	return result
}

// Execute runs a tool by name with the given input
func (tr *ToolRegistry) Execute(ctx context.Context, toolName string, input ToolInput) (*ToolOutput, error) {
	tool := tr.Get(toolName)
	if tool == nil {
		return &ToolOutput{
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", toolName),
		}, fmt.Errorf("tool not found: %s", toolName)
	}

	log.Printf("🔧 Executing tool: %s with args: %v", toolName, input.Args)

	result, err := tool.Handler(ctx, input.Args)
	if err != nil {
		return &ToolOutput{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	return &ToolOutput{
		Success: true,
		Data:    result,
	}, nil
}

// --- BUILT-IN TOOL EXAMPLES (ready to wire in) ---

// WebSearchTool performs semantic web search
func WebSearchTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "web_search",
		Description: "Search the web for current information",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (1-10)",
					"default":     5,
				},
			},
			"required": []string{"query"},
		},
		Handler: webSearchHandler,
	}
}

func webSearchHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// TODO: Integrate with your preferred search provider (Tavily, Exa, DuckDuckGo, etc.)
	// For now, this is a placeholder
	log.Printf("🔍 Web search would execute: %s", query)

	return map[string]interface{}{
		"query":   query,
		"results": []string{"Result 1", "Result 2"}, // Replace with actual API call
	}, nil
}

// MathTool performs calculations
func MathTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "calculate",
		Description: "Perform mathematical calculations",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "Math expression (e.g., '2 + 2 * 3')",
				},
			},
			"required": []string{"expression"},
		},
		Handler: mathHandler,
	}
}

func mathHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	expression, ok := args["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("expression must be a string")
	}

	// TODO: Use a safe expression evaluator (e.g., expr library)
	log.Printf("🧮 Math calculation: %s", expression)

	return map[string]interface{}{
		"expression": expression,
		"result":     0, // Replace with actual calculation
	}, nil
}

// FileFetchTool retrieves file contents
func FileFetchTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "fetch_file",
		Description: "Read contents from a local file",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "File path",
				},
			},
			"required": []string{"path"},
		},
		Handler: fileFetchHandler,
	}
}

func fileFetchHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	// TODO: Implement safe file read with permission checks
	log.Printf("📄 Fetch file: %s", path)

	return map[string]interface{}{
		"path":     path,
		"contents": "File contents here",
	}, nil
}

// HTTPCallTool makes HTTP requests
func HTTPCallTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "http_call",
		Description: "Make HTTP requests to external APIs",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to call",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"GET", "POST", "PUT", "DELETE"},
					"description": "HTTP method",
					"default":     "GET",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "Request body (for POST/PUT)",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "HTTP headers",
				},
			},
			"required": []string{"url"},
		},
		Handler: httpCallHandler,
	}
}

func httpCallHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	method := "GET"
	if m, exists := args["method"]; exists {
		method = m.(string)
	}

	// TODO: Implement HTTP call with context timeout and error handling
	log.Printf("📡 HTTP %s: %s", method, url)

	return map[string]interface{}{
		"url":    url,
		"method": method,
		"status": 200,
		"body":   "Response body",
	}, nil
}

// JSONParseTool parses JSON strings
func JSONParseTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "parse_json",
		Description: "Parse and validate JSON strings",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"json_string": map[string]interface{}{
					"type":        "string",
					"description": "JSON string to parse",
				},
			},
			"required": []string{"json_string"},
		},
		Handler: jsonParseHandler,
	}
}

func jsonParseHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	jsonStr, ok := args["json_string"].(string)
	if !ok {
		return nil, fmt.Errorf("json_string must be a string")
	}

	var parsed interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	log.Printf("✅ JSON parsed successfully")
	return parsed, nil
}

// CustomDatabaseQueryTool queries your backend database
func CustomDatabaseQueryTool() *ToolDefinition {
	return &ToolDefinition{
		Name:        "query_database",
		Description: "Execute custom database queries against your backend",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Database query",
				},
				"params": map[string]interface{}{
					"type":        "object",
					"description": "Query parameters",
				},
			},
			"required": []string{"query"},
		},
		Handler: customDatabaseHandler,
	}
}

func customDatabaseHandler(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	// TODO: Implement your database call here
	log.Printf("📊 Database query: %s", query)

	return map[string]interface{}{
		"query":   query,
		"results": []map[string]interface{}{}, // Replace with actual results
	}, nil
}
