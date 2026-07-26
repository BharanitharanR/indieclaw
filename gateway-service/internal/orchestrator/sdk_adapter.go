package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LoadMCPToolsViaSDK spins up a stdio MCP server using the official SDK,
// fetches all tools cleanly without buffer limitations, and returns them
// in a format your existing architecture already understands.
func LoadMCPToolsViaSDK(ctx context.Context, command string, args []string) ([]*ToolDefinition, func(), error) {
	// 1. Initialize official client implementation
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "indieclaw-orchestrator",
		Version: "1.0.0",
	}, nil)

	// 2. Configure standard command subprocess transport
	transport := &mcp.CommandTransport{
		Command: exec.Command(command, args...),
	}

	// 3. Connect (handles initialization handshake automatically)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("sdk connection failed: %w", err)
	}

	// Cleanup closure to close session process when done
	closer := func() {
		_ = session.Close()
	}

	// 4. Fetch all available tools using the SDK session
	toolsResult, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		closer()
		return nil, nil, fmt.Errorf("sdk failed to list tools: %w", err)
	}

	var definitions []*ToolDefinition
	for _, t := range toolsResult.Tools {
		toolName := t.Name
		desc := t.Description

		// Safely convert t.InputSchema (which is `any`) to map[string]interface{}
		var schema map[string]interface{}
		if schemaMap, ok := t.InputSchema.(map[string]interface{}); ok {
			schema = schemaMap
		} else {
			// Fallback: marshal and unmarshal if it's a struct representation
			if rawBytes, err := json.Marshal(t.InputSchema); err == nil {
				_ = json.Unmarshal(rawBytes, &schema)
			}
			if schema == nil {
				schema = make(map[string]interface{})
			}
		}

		handler := func(activeSession *mcp.ClientSession, name string) ToolHandler {
			return func(c context.Context, arguments map[string]interface{}) (interface{}, error) {
				res, callErr := activeSession.CallTool(c, &mcp.CallToolParams{
					Name:      name,
					Arguments: arguments,
				})
				if callErr != nil {
					return nil, callErr
				}

				var output string
				for _, content := range res.Content {
					if textContent, ok := content.(*mcp.TextContent); ok {
						output += textContent.Text
					}
				}

				return map[string]interface{}{
					"status": "success",
					"result": output,
				}, nil
			}
		}(session, toolName)

		definitions = append(definitions, &ToolDefinition{
			Name:        toolName,
			Description: desc,
			InputSchema: schema,
			Handler:     handler,
		})
	}
	return definitions, closer, nil
}
