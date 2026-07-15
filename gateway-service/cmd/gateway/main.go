package main

import (
	"context"
	"gateway-service/internal/mcp"
	"gateway-service/internal/service"
	"log"
	"net/http"
	"time"
)

func main() {
	ollamaURL := "http://127.0.0.1:11434"
	defaultModel := "gemma4:12b"

	// 1. Setup Context for MCP initialization handshake
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 2. Initialize MCP Tool Registry
	registry := setupMCPRegistry(ctx)

	// 3. Pass the registry to OllamaService so it can query & execute MCP tools
	ollamaSvc := service.NewOllamaService(ollamaURL, defaultModel, registry)

	// 4. Register HTTP Routes & Start Server
	mux := http.NewServeMux()
	ollamaSvc.RegisterRoutes(mux)

	log.Println("🚀 Gateway running on http://127.0.0.1:8080")
	if err := http.ListenAndServe("127.0.0.1:8080", mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}

func setupMCPRegistry(ctx context.Context) *mcp.ToolRegistry {
	registry := mcp.NewToolRegistry()

	log.Println("🔌 Initializing MCP Servers...")

	// 1. Filesystem MCP
	if err := registry.RegisterServer(ctx, "fs", "npx", "-y", "@modelcontextprotocol/server-filesystem", "/Users/bharani/projects"); err != nil {
		log.Printf("⚠️ Failed to load Filesystem MCP: %v", err)
	}

	// Register DuckDuckGo MCP Web Search (No API Key Required)
	// Using `uvx` executes the package directly without global installation
	err := registry.RegisterServer(ctx, "ddg-search", "uvx", "duckduckgo-mcp-server")
	if err != nil {
		log.Printf("⚠️ Failed to load DuckDuckGo MCP Server: %v", err)
		log.Println("💡 Tip: Make sure `uv` is installed (`brew install uv`) or use `python3` instead.")
	}

	// 3. GitHub Remote API Server
	// Note: Requires GITHUB_PERSONAL_ACCESS_TOKEN in system environment
	// if err := registry.RegisterServer(ctx, "github", "npx", "-y", "@modelcontextprotocol/server-github"); err != nil {
	//	log.Printf("⚠️ Failed to load GitHub MCP: %v", err)
	// }

	allTools := registry.GetAllTools()
	// Print out the tool names and descriptions
	for i, tool := range allTools {
		log.Printf("  %2d. %-20s - %s", i+1, tool.Name, tool.Description)
	}
	log.Printf("✅ Ready! %d total MCP tools loaded into Gateway Registry.", len(allTools))

	return registry
}
