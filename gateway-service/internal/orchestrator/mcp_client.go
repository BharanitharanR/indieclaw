package orchestrator

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MCPMessage represents a JSON-RPC 2.0 message
type MCPMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// Add these structures to handle your config JSON format
type MCPServerConfig struct {
	Name string `json:"name"`
	Type string `json:"type"` // "http" or "stdio"
	URL  string `json:"url"`  // Used for HTTP address or Stdio command string
}

type MCPRootConfig struct {
	Servers []MCPServerConfig `json:"servers"`
}

// MCPError represents a JSON-RPC error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPTool represents a tool from an MCP server
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPListToolsResponse represents the response from list_tools
type MCPListToolsResponse struct {
	Tools []MCPTool `json:"tools"`
}

// MCPCallToolRequest represents params for tool_call
type MCPCallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// MCPCallToolResponse represents the response from tool_call
type MCPCallToolResponse struct {
	Content []MCPContent `json:"content"`
}

// MCPContent represents content in a tool response
type MCPContent struct {
	Type string `json:"type"` // "text", "image", etc.
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MIME string `json:"mimeType,omitempty"`
}

// MCPClient manages connection to an MCP server

type MCPClient struct {
	serverURL    string
	isHTTP       bool // Added
	httpClient   *http.Client
	cmd          *exec.Cmd      // Added
	stdin        io.WriteCloser // Added
	requestID    int64
	pendingMu    sync.Mutex            // Added
	pendingReqs  map[int64]chan []byte // Added
	capabilities map[string]interface{}
	initalized   bool
}

// Replace your NewMCPClient constructor with this clean version
func NewMCPClient(serverType string, connectionString string) *MCPClient {
	isHTTP := strings.ToLower(serverType) == "http"

	client := &MCPClient{
		serverURL:   connectionString,
		isHTTP:      isHTTP,
		pendingReqs: make(map[int64]chan []byte),
	}

	if isHTTP {
		client.httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
		return client
	}

	// Stdio Path: Parses the connection string as a shell execution command
	parts := strings.Fields(connectionString)
	if len(parts) == 0 {
		log.Printf("❌ Stdio command cannot be blank")
		return client
	}

	cmdName := parts[0]
	cmdArgs := parts[1:]

	cmd := exec.Command(cmdName, cmdArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdin pipe: %v", err)
		return client
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("❌ Failed to create stdout pipe: %v", err)
		return client
	}
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start stdio process: %v", err)
		return client
	}

	client.cmd = cmd
	client.stdin = stdin

	// Replace the scanner loop in NewMCPClient with this bufio.Reader loop:
	go func(stdout io.Reader) {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("❌ Stdio read error: %v", err)
				}
				break
			}

			var tracker struct {
				ID int64 `json:"id"`
			}
			if err := json.Unmarshal(line, &tracker); err != nil {
				continue
			}

			client.pendingMu.Lock()
			ch, exists := client.pendingReqs[tracker.ID]
			if exists {
				ch <- append([]byte(nil), line...)
				delete(client.pendingReqs, tracker.ID)
			}
			client.pendingMu.Unlock()
		}
	}(stdout)

	return client
}

// Initialize performs the MCP handshake
func (mc *MCPClient) Initialize(ctx context.Context) error {
	log.Printf("🔌 Initializing MCP client for: %s", mc.serverURL)

	// Step 1: Send initialize request
	initMsg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "initialize",
		Params: json.RawMessage([]byte(`{
			"protocolVersion": "2024-11-05",
			"capabilities": {
				"tools": {}
			},
			"clientInfo": {
				"name": "indieclaw-orchestrator",
				"version": "1.0.0"
			}
		}`)),
		ID: mc.nextRequestID(), // 💡 Use atomic generator instead of hardcoded 1
	}

	resp, err := mc.sendRequest(ctx, initMsg)
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Extract capabilities from response
	var initResp struct {
		Result struct {
			Capabilities map[string]interface{} `json:"capabilities"`
			ServerInfo   struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"serverInfo"`
		} `json:"result"`
	}

	if err := json.Unmarshal(resp, &initResp); err != nil {
		return fmt.Errorf("failed to parse initialize response: %w", err)
	}

	mc.capabilities = initResp.Result.Capabilities
	mc.initalized = true
	mc.capabilities = initResp.Result.Capabilities
	mc.initalized = true

	// ADD THIS: Send required protocol handshake validation notification
	notifMsg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}
	notifJSON, _ := json.Marshal(notifMsg)
	if !mc.isHTTP && mc.stdin != nil {
		_, _ = mc.stdin.Write(append(notifJSON, '\n'))
	}

	log.Printf("✅ MCP initialized: %s v%s", initResp.Result.ServerInfo.Name, initResp.Result.ServerInfo.Version)
	return nil
}

// ListTools fetches available tools from the MCP server
func (mc *MCPClient) ListTools(ctx context.Context) ([]MCPTool, error) {
	if !mc.initalized {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	log.Printf("📋 Fetching tools from MCP server...")

	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/list",
		Params:  json.RawMessage([]byte(`{}`)),
		ID:      mc.nextRequestID(),
	}

	resp, err := mc.sendRequest(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("list_tools request failed: %w", err)
	}

	var toolsResp struct {
		Result MCPListToolsResponse `json:"result"`
	}

	if err := json.Unmarshal(resp, &toolsResp); err != nil {
		return nil, fmt.Errorf("failed to parse tools response: %w", err)
	}

	log.Printf("📦 Found %d tools in MCP server", len(toolsResp.Result.Tools))
	for _, tool := range toolsResp.Result.Tools {
		log.Printf("  - %s: %s", tool.Name, tool.Description)
	}

	return toolsResp.Result.Tools, nil
}

// CallTool invokes a tool on the MCP server
func (mc *MCPClient) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (string, error) {
	if !mc.initalized {
		return "", fmt.Errorf("MCP client not initialized")
	}

	log.Printf("🔧 Calling MCP tool: %s with args: %v", toolName, arguments)

	toolCallReq := MCPCallToolRequest{
		Name:      toolName,
		Arguments: arguments,
	}

	paramsJSON, _ := json.Marshal(toolCallReq)

	msg := MCPMessage{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params:  json.RawMessage(paramsJSON),
		ID:      mc.nextRequestID(),
	}

	resp, err := mc.sendRequest(ctx, msg)
	if err != nil {
		return "", fmt.Errorf("tool call request failed: %w", err)
	}

	var toolResp struct {
		Result MCPCallToolResponse `json:"result"`
		Error  *MCPError           `json:"error"`
	}

	if err := json.Unmarshal(resp, &toolResp); err != nil {
		return "", fmt.Errorf("failed to parse tool response: %w", err)
	}

	if toolResp.Error != nil {
		return "", fmt.Errorf("tool execution error: %s", toolResp.Error.Message)
	}

	// Extract text content from response
	var result strings.Builder
	for _, content := range toolResp.Result.Content {
		if content.Type == "text" {
			result.WriteString(content.Text)
		}
	}

	log.Printf("✅ MCP tool response received")
	return result.String(), nil
}

func (mc *MCPClient) sendRequest(ctx context.Context, msg MCPMessage) (json.RawMessage, error) {
	msgJSON, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	var id64 int64
	switch v := msg.ID.(type) {
	case int:
		id64 = int64(v)
	case int64:
		id64 = v
	case float64:
		id64 = int64(v)
	}

	if !mc.isHTTP {
		return mc.sendStdioRequest(ctx, id64, msgJSON)
	}

	log.Printf("📤 Sending to MCP: %s", string(msgJSON))

	req, err := http.NewRequestWithContext(ctx, "POST", mc.serverURL, bytes.NewReader(msgJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	httpResp, err := mc.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK && httpResp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("MCP server returned status %d: %s", httpResp.StatusCode, string(body))
	}

	bodyStr := string(body)
	finalJSON := body
	if strings.Contains(bodyStr, "data: ") {
		lines := strings.Split(bodyStr, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "data: ") {
				finalJSON = []byte(strings.TrimPrefix(line, "data: "))
				break
			}
		}
	}

	log.Printf("📥 Received from MCP: %s", string(finalJSON))

	var msgResp MCPMessage
	if err := json.Unmarshal(finalJSON, &msgResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if msgResp.Error != nil {
		return nil, fmt.Errorf("MCP error (%d): %s", msgResp.Error.Code, msgResp.Error.Message)
	}

	return msgResp.Result, nil
}

func (mc *MCPClient) sendStdioRequest(ctx context.Context, id int64, msgJSON []byte) (json.RawMessage, error) {
	log.Printf("📤 Sending via Stdio: %s", string(msgJSON))

	resCh := make(chan []byte, 1)
	mc.pendingMu.Lock()
	mc.pendingReqs[id] = resCh
	mc.pendingMu.Unlock()

	// Write payload followed by the mandatory protocol line break delimiter
	if _, err := mc.stdin.Write(append(msgJSON, '\n')); err != nil {
		mc.pendingMu.Lock()
		delete(mc.pendingReqs, id)
		mc.pendingMu.Unlock()
		return nil, fmt.Errorf("failed writing to stdin process: %w", err)
	}

	select {
	case <-ctx.Done():
		mc.pendingMu.Lock()
		delete(mc.pendingReqs, id)
		mc.pendingMu.Unlock()
		return nil, ctx.Err()
	case rawResponse := <-resCh:
		log.Printf("📥 Received via Stdio: %s", string(rawResponse))

		var msgResp MCPMessage
		if err := json.Unmarshal(rawResponse, &msgResp); err != nil {
			return nil, fmt.Errorf("failed parsing stdout stream: %w", err)
		}

		if msgResp.Error != nil {
			return nil, fmt.Errorf("MCP error (%d): %s", msgResp.Error.Code, msgResp.Error.Message)
		}

		return msgResp.Result, nil
	}
}

// nextRequestID increments and returns the next request ID safely
func (mc *MCPClient) nextRequestID() int64 {
	return atomic.AddInt64(&mc.requestID, 1)
}

// ============= MCP TOOL ADAPTER: Bridge MCP tools to ToolRegistry =============

// MCPToolAdapter wraps an MCP server as a tool provider
type MCPToolAdapter struct {
	client *MCPClient
	tools  map[string]*ToolDefinition
}

// Replace your NewMCPToolAdapter constructor with this matching version
func NewMCPToolAdapter(serverType string, connectionString string) (*MCPToolAdapter, error) {
	client := NewMCPClient(serverType, connectionString)

	return &MCPToolAdapter{
		client: client,
		tools:  make(map[string]*ToolDefinition),
	}, nil
}

// Initialize and fetch tools from the MCP server
func (adapter *MCPToolAdapter) Initialize(ctx context.Context) error {
	// Step 1: Initialize MCP handshake
	if err := adapter.client.Initialize(ctx); err != nil {
		return fmt.Errorf("MCP initialization failed: %w", err)
	}

	// Step 2: Fetch tools from MCP server
	mcpTools, err := adapter.client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch MCP tools: %w", err)
	}

	// Step 3: Convert MCP tools to our ToolDefinition format
	for _, mcpTool := range mcpTools {
		// Create a closure to capture the tool name
		toolName := mcpTool.Name
		toolDesc := mcpTool.Description
		toolSchema := mcpTool.InputSchema

		handler := func(mcpToolName string) ToolHandler {
			return func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
				result, err := adapter.client.CallTool(ctx, mcpToolName, args)
				if err != nil {
					return nil, err
				}
				return map[string]interface{}{
					"status": "success",
					"result": result,
				}, nil
			}
		}(toolName)

		toolDef := &ToolDefinition{
			Name:        toolName,
			Description: toolDesc,
			InputSchema: toolSchema,
			Handler:     handler,
		}

		adapter.tools[toolName] = toolDef
		log.Printf("✅ MCP tool wrapped: %s", toolName)
	}

	return nil
}

// GetToolDefinitions returns all MCP tools as ToolDefinitions
func (adapter *MCPToolAdapter) GetToolDefinitions() []*ToolDefinition {
	var defs []*ToolDefinition
	for _, tool := range adapter.tools {
		defs = append(defs, tool)
	}
	return defs
}

// RegisterToRegistry registers all MCP tools to a ToolRegistry
func (adapter *MCPToolAdapter) RegisterToRegistry(registry *ToolRegistry) error {
	for _, tool := range adapter.tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register MCP tool %s: %w", tool.Name, err)
		}
	}
	return nil
}

// ============= MULTI-MCP SERVER MANAGER =============

// MCPServerManager coordinates multiple active internal MCP client instances
// MCPServerManager coordinates multiple active internal MCP client instances
type MCPServerManager struct {
	mu       sync.RWMutex
	adapters map[string]*MCPToolAdapter // Updated to fix the compilation error
}

// NewMCPServerManager bootstraps a fresh server coordination map
func NewMCPServerManager() *MCPServerManager {
	return &MCPServerManager{
		adapters: make(map[string]*MCPToolAdapter), // Updated
	}
}

// AddServer hooks up a new connection type and runs the handshake execution sequence
// AddServer hooks up a new connection type and runs the handshake execution sequence
func (sm *MCPServerManager) AddServerOld(ctx context.Context, name string, serverType string, connectionString string) error {
	// Initialize the custom client using the updated multi-transport parameters
	client := NewMCPClient(serverType, connectionString)

	// Execute initialization handshake protocol natively over the selected medium
	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize client %s: %w", name, err)
	}

	// Fetch available tool sets from the backend instance
	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return fmt.Errorf("failed listing tools for %s: %w", name, err)
	}

	// Construct the adapter expected by your tracking architecture
	adapter := &MCPToolAdapter{
		client: client,
		tools:  make(map[string]*ToolDefinition),
	}

	// Map the raw tool sets into your adapter structure
	for _, tool := range mcpTools {
		adapter.tools[tool.Name] = &ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
		}
	}

	// Save to the adapters tracking map safely
	sm.mu.Lock()
	sm.adapters[name] = adapter
	sm.mu.Unlock()

	return nil
}

// AddServer hooks up a new connection type and runs the handshake execution sequence using the official SDK for Stdio
func (sm *MCPServerManager) AddServer(ctx context.Context, name string, serverType string, connectionString string) error {
	var definitions []*ToolDefinition
	var closer func()
	var err error

	// Route Stdio commands through the official SDK loader to avoid buffer/JSON-RPC parsing bugs
	if strings.ToLower(serverType) == "stdio" {
		parts := strings.Fields(connectionString)
		if len(parts) == 0 {
			return fmt.Errorf("stdio command cannot be blank for server %s", name)
		}
		cmdName := parts[0]
		cmdArgs := parts[1:]

		// Call the SDK loader we created in sdk_adapter.go
		definitions, closer, err = LoadMCPToolsViaSDK(ctx, cmdName, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed loading via SDK for %s: %w", name, err)
		}
	} else {
		// Fallback or handle HTTP transport using your original logic
		client := NewMCPClient(serverType, connectionString)
		if err := client.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize client %s: %w", name, err)
		}
		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed listing tools for %s: %w", name, err)
		}

		// Map standard HTTP tools
		for _, tool := range mcpTools {
			definitions = append(definitions, &ToolDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			})
		}
	}

	// Construct the standard adapter expected by your tracking architecture
	adapter := &MCPToolAdapter{
		tools: make(map[string]*ToolDefinition),
	}

	// Populate the adapter's tool map
	for _, toolDef := range definitions {
		adapter.tools[toolDef.Name] = toolDef
	}

	// Save to the adapters tracking map safely
	sm.mu.Lock()
	sm.adapters[name] = adapter
	sm.mu.Unlock()

	log.Printf("📦 Successfully registered %d tools from server: %s", len(definitions), name)

	// Optional: if you want to store or handle the closer function for cleanup later,
	// you can attach it to a cleanup tracker array in your MCPServerManager struct.
	_ = closer

	return nil
}

// RegisterAllToRegistry registers all MCP server tools to a registry
func (m *MCPServerManager) RegisterAllToRegistry(registry *ToolRegistry) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for serverName, adapter := range m.adapters {
		log.Printf("📦 Registering tools from: %s", serverName)
		if err := adapter.RegisterToRegistry(registry); err != nil {
			return fmt.Errorf("failed to register tools from %s: %w", serverName, err)
		}
	}

	return nil
}

// GetAdapter returns a specific MCP adapter
func (m *MCPServerManager) GetAdapter(serverName string) *MCPToolAdapter {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adapters[serverName]
}

// ListAllTools returns all tools from all MCP servers
func (m *MCPServerManager) ListAllTools() map[string][]*ToolDefinition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]*ToolDefinition)
	for serverName, adapter := range m.adapters {
		result[serverName] = adapter.GetToolDefinitions()
	}
	return result
}

// ============= HELPER: Stream-based initialization (for SSE) =============

// InitializeWithSSE handles Server-Sent Events stream from MCP server
// Use this if your MCP server uses streaming protocol instead of HTTP POST
func (mc *MCPClient) InitializeWithSSE(ctx context.Context) error {
	log.Printf("🔌 Initializing MCP client with SSE stream: %s", mc.serverURL)

	// Create SSE request
	req, err := http.NewRequestWithContext(ctx, "GET", mc.serverURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create SSE request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream")

	httpResp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connection failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE server returned status %d", httpResp.StatusCode)
	}

	// Parse SSE events
	scanner := bufio.NewScanner(httpResp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse SSE event
		if strings.HasPrefix(line, "data: ") {
			eventData := strings.TrimPrefix(line, "data: ")

			var msg MCPMessage
			if err := json.Unmarshal([]byte(eventData), &msg); err != nil {
				log.Printf("Failed to parse SSE event: %v", err)
				continue
			}

			// Handle initialize response
			if msg.Method == "initialized" {
				mc.initalized = true
				log.Printf("✅ MCP initialized via SSE")
				return nil
			}
		}
	}

	return scanner.Err()
}
