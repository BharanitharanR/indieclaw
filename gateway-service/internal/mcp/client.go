package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

)

type MCPClientManager struct {
	mcpClient client.MCPClient
	mcpTools  []mcp.Tool
}

// NewStdioClient launches an external MCP server binary via stdio
func NewStdioClient(ctx context.Context, command string, args ...string) (*MCPClientManager, error) {
	c, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create stdio client: %w", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "go-gateway-client",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("mcp handshake failed: %w", err)
	}

	log.Printf("✅ Connected to MCP Server: %s %v", command, args)
	return &MCPClientManager{mcpClient: c}, nil
}

// FetchTools lists all tools available on the MCP server
func (m *MCPClientManager) FetchTools(ctx context.Context) ([]mcp.Tool, error) {
	req := mcp.ListToolsRequest{}
	res, err := m.mcpClient.ListTools(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	return res.Tools, nil
}

// ExecuteTool calls a specific tool on the MCP server
func (m *MCPClientManager) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = arguments

	res, err := m.mcpClient.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("tool execution error: %w", err)
	}

	if res.IsError {
		return "", fmt.Errorf("mcp tool returned an error response")
	}

	var output string
	for _, content := range res.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			output += textContent.Text + "\n"
		}
	}

	return output, nil
}

// Close terminates the connection to the MCP server
func (m *MCPClientManager) Close() error {
	return m.mcpClient.Close()
}
