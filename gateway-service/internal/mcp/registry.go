package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
)

type ToolRegistry struct {
	mu      sync.RWMutex
	clients map[string]*MCPClientManager // maps toolName -> MCPClient
	tools   []mcp.Tool                   // combined list of all tools
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		clients: make(map[string]*MCPClientManager),
		tools:   make([]mcp.Tool, 0),
	}
}

// RegisterServer connects to an MCP server and registers all its tools
func (r *ToolRegistry) RegisterServer(ctx context.Context, name string, command string, args ...string) error {
	client, err := NewStdioClient(ctx, command, args...)
	if err != nil {
		return fmt.Errorf("failed to register server %s: %w", name, err)
	}

	serverTools, err := client.FetchTools(ctx)
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to fetch tools for %s: %w", name, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tool := range serverTools {
		r.clients[tool.Name] = client
		r.tools = append(r.tools, tool)
	}

	return nil
}

// GetAllTools returns all registered MCP tools to send to the LLM
func (r *ToolRegistry) GetAllTools() []mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tools
}

// Execute looks up which client owns the tool and executes it
func (r *ToolRegistry) Execute(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	r.mu.RLock()
	client, exists := r.clients[toolName]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool '%s' not found in registry", toolName)
	}

	return client.ExecuteTool(ctx, toolName, args)
}
