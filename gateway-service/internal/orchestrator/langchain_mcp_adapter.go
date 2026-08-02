package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tmc/langchaingo/tools"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
)

// MCPConfig represents the structure of your MCP servers JSON configuration file
type MCPConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

type MCPServerConfig struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`    // "stdio", "http", "sse", or "streamable-http"
	URL     string            `json:"url"`     // Executable path or endpoint URL
	Args    []string          `json:"args"`    // Optional arguments for stdio
	Headers map[string]string `json:"headers"` // Optional custom headers (e.g., Authorization bearer tokens)
}

// LoadMCPToolsFromConfig connects to MCP servers and wraps their tools into LangChain tools
func LoadMCPToolsFromConfig(ctx context.Context, configPath string) ([]tools.Tool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config file %s: %w", configPath, err)
	}

	var config MCPConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config JSON: %w", err)
	}

	var allLangChainTools []tools.Tool

	for _, srv := range config.Servers {
		log.Printf("🔌 Connecting to MCP Server: %s [Type: %s, Endpoint/Path: %s]", srv.Name, srv.Type, srv.URL)

		var mcpClient *client.Client
		var clientErr error

		switch srv.Type {
		case "stdio":
			mcpClient, clientErr = client.NewStdioMCPClient(srv.URL, os.Environ(), srv.Args...)
		case "sse", "http", "streamable-http", "httpstreaming", "http-streamable":
			mcpClient, clientErr = NewAuthenticatedStreamableClient(srv.URL, srv.Headers)
		default:
			log.Printf("⚠️ Unknown MCP server transport type '%s' for server %s", srv.Type, srv.Name)
			continue
		}

		if clientErr != nil {
			log.Printf("❌ Failed to initialize MCP client for %s: %v", srv.Name, clientErr)
			continue
		}

		// Initialize session
		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    "gateway-service",
			Version: "1.0.0",
		}

		_, err = mcpClient.Initialize(ctx, initRequest)
		if err != nil {
			log.Printf("❌ Failed to initialize MCP session for %s: %v", srv.Name, err)
			_ = mcpClient.Close()
			continue
		}

		// List tools available from the MCP server
		toolsResult, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			log.Printf("❌ Failed to list tools from MCP server %s: %v", srv.Name, err)
			_ = mcpClient.Close()
			continue
		}

		// Wrap each MCP tool into a LangChain-compatible tool definition via our bridge
		for _, mcpTool := range toolsResult.Tools {
			toolName := mcpTool.Name
			toolDesc := mcpTool.Description

			var schemaMap map[string]interface{}
			schemaBytes, err := json.Marshal(mcpTool.InputSchema)
			if err == nil {
				_ = json.Unmarshal(schemaBytes, &schemaMap)
			}
			if schemaMap == nil {
				schemaMap = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
			}

			toolDef := &ToolDefinition{
				Name:        toolName,
				Description: toolDesc,
				InputSchema: schemaMap,
				Handler: func(callCtx context.Context, args map[string]interface{}) (interface{}, error) {
					callReq := mcp.CallToolRequest{}
					callReq.Params.Name = toolName
					callReq.Params.Arguments = args

					res, callErr := mcpClient.CallTool(callCtx, callReq)
					if callErr != nil {
						return nil, callErr
					}

					var extractedText string
					for _, content := range res.Content {
						if textContent, ok := content.(mcp.TextContent); ok {
							extractedText += textContent.Text
						} else {
							contentBytes, err := json.Marshal(content)
							if err == nil {
								extractedText += string(contentBytes)
							} else {
								extractedText += fmt.Sprintf("%v", content)
							}
						}
					}

					if extractedText == "" {
						extractedText = fmt.Sprintf("%v", res)
					}

					return extractedText, nil
				},
			}

			allLangChainTools = append(allLangChainTools, NewLangChainToolBridge(toolDef))
		}

		log.Printf("✅ Successfully loaded %d tools from MCP server: %s", len(toolsResult.Tools), srv.Name)
	}

	return allLangChainTools, nil
}

// NewAuthenticatedStreamableClient creates a streamable HTTP MCP client with custom headers
func NewAuthenticatedStreamableClient(baseURL string, headers map[string]string) (*client.Client, error) {
	var opts []transport.StreamableHTTPCOption

	if len(headers) > 0 {
		customHTTPClient := &http.Client{
			Transport: &headerRoundTripper{
				headers: headers,
				base:    http.DefaultTransport,
			},
		}
		opts = append(opts, transport.WithHTTPBasicClient(customHTTPClient))
	}

	trans, err := transport.NewStreamableHTTP(baseURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create streamable HTTP transport: %w", err)
	}

	clientOptions := []client.ClientOption{}
	if sessionID := trans.GetSessionId(); sessionID != "" {
		clientOptions = append(clientOptions, client.WithSession())
	}

	return client.NewClient(trans, clientOptions...), nil
}

// headerRoundTripper injects custom headers into outgoing requests
type headerRoundTripper struct {
	headers map[string]string
	base    http.RoundTripper
}

func (h *headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	reqClone := req.Clone(req.Context())
	for k, v := range h.headers {
		reqClone.Header.Set(k, v)
	}
	baseTransport := h.base
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	return baseTransport.RoundTrip(reqClone)
}
