package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	// Adjusted package names to match your project imports

	initdb "gateway-service/cmd/init_db"
	"gateway-service/internal/orchestrator"
	v1 "gateway-service/proto/gateway/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	v1.UnimplementedGatewayServiceServer
	workflow       *orchestrator.AgentWorkflow
	qdrantClient   *qdrant.Client // Added to hold the live client instance
	textModel      string
	visionModel    string
	embeddingModel string
	// Thread-safe Graph Memory Map
	graphMu          sync.RWMutex
	activeSessions   map[string]*SessionNode
	toolRegistry     *orchestrator.ToolRegistry
	mcpServerManager *orchestrator.MCPServerManager // NEW: MCP server manager
}

func (s *server) lookupOrInitializeSession(ctx context.Context, userID, query string) (string, string) {
	// 1. Check in-memory state graph first for rapid recency matching
	s.graphMu.RLock()
	activeNode, exists := s.activeSessions[userID]
	s.graphMu.RUnlock()

	if exists && time.Since(activeNode.LastSeen) < 5*time.Minute {
		log.Printf("🔗 Graph Context Link: User %s active within 5m. Extending session: %s", userID, activeNode.SessionID)
		return activeNode.SessionID, activeNode.PreviousVectorID
	}

	// 2. Fallback to Semantic Vector Store check if memory window expired or missing
	if s.qdrantClient != nil {
		queryVector, err := orchestrator.EmbedQuery(ctx, s.embeddingModel, query)
		if err == nil {
			limitVal := uint64(1)
			searchResult, err := s.qdrantClient.Query(ctx, &qdrant.QueryPoints{
				CollectionName: "chat_history",
				Query:          qdrant.NewQuery(queryVector...),
				Limit:          &limitVal,
				WithPayload:    qdrant.NewWithPayload(true),
			})

			if err == nil && len(searchResult) > 0 {
				topMatch := searchResult[0]
				// If semantic match is confidently established
				if topMatch.Score > 0.80 {
					payload := topMatch.Payload
					sessID := payload["session_id"].GetStringValue()

					// Reconstruct point ID string from Qdrant variant
					var parentPointID string
					if topMatch.Id != nil && topMatch.Id.GetUuid() != "" {
						parentPointID = topMatch.Id.GetUuid()
					}

					log.Printf("🎯 Vector Match! Resuming historic session ID: %s (Parent Vector ID: %s)", sessID, parentPointID)
					return sessID, parentPointID
				}
			}
		}
	}

	// 3. True Drift Context: Spin up entirely new session markers
	newSessionID := uuid.NewString()
	log.Printf("ℹ️ Initializing completely fresh session track: %s", newSessionID)
	return newSessionID, ""
}

type SessionNode struct {
	SessionID        string
	PreviousVectorID string // Tracks the last saved Qdrant point UUID string
	LastSeen         time.Time
}

// updateGraphContext safely registers or updates the latest edge point for a specific user
func (s *server) updateGraphContext(userID, sessionID, vectorID string) {
	s.graphMu.Lock()
	defer s.graphMu.Unlock()

	if s.activeSessions == nil {
		s.activeSessions = make(map[string]*SessionNode)
	}

	s.activeSessions[userID] = &SessionNode{
		SessionID:        sessionID,
		PreviousVectorID: vectorID,
		LastSeen:         time.Now(),
	}
}

func main() {
	textModel := os.Getenv("TEXT_MODEL")
	if textModel == "" {
		textModel = "qwen3:8b"
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel == "" {
		visionModel = "gemma4:e2b"
	}

	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "nomic-embed-text"
	}

	ctx := context.Background()
	wf, err := orchestrator.NewAgentWorkflow(ctx, textModel, visionModel)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	s := grpc.NewServer()

	srv := &server{
		workflow:         wf,
		textModel:        textModel,
		visionModel:      visionModel,
		embeddingModel:   embeddingModel,
		activeSessions:   make(map[string]*SessionNode),
		mcpServerManager: orchestrator.NewMCPServerManager(), // NEW: Initialize MCP manager
	}

	// 🎯 Initialize tool registry
	srv.initializeTools()

	// 🌐 NEW: Initialize MCP servers from environment or config
	initializeMCPServers(ctx, srv)

	// Fire up Qdrant initialization concurrently...
	go func() {
		log.Println("🔄 Background thread: Initializing local Qdrant collection...")

		for {
			client, err := initdb.InitializeVectorStore()
			if err != nil {
				log.Printf("❌ Qdrant not ready yet (%v). Retrying in 3 seconds...", err)
				time.Sleep(3 * time.Second)
				continue
			}

			srv.qdrantClient = client
			log.Println("✅ Background thread: Qdrant client connected and successfully wired.")
			break
		}
	}()

	v1.RegisterGatewayServiceServer(s, srv)
	reflection.Register(s)

	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on :9000: %v", err)
	}

	log.Printf("🚀 Orchestrator running on :9000 [Text: %s, Vision: %s, Tools: ENABLED, MCP: %d servers]",
		textModel, visionModel, len(srv.mcpServerManager.ListAllTools()))
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// ============= STEP 3: Add MCP initialization function =============

// initializeMCPServers sets up MCP server connections from environment or config
func initializeMCPServers(ctx context.Context, srv *server) {
	// Method 1: From environment variables (e.g., MCP_SERVERS=gmail:http://localhost:3001,notion:http://localhost:3002)
	mcpServersEnv := os.Getenv("MCP_SERVERS")
	if mcpServersEnv != "" {
		log.Println("🔌 Loading MCP servers from MCP_SERVERS environment variable...")
		servers := strings.Split(mcpServersEnv, ",")
		for _, serverPair := range servers {
			parts := strings.Split(strings.TrimSpace(serverPair), ":")
			if len(parts) != 3 {
				log.Printf("⚠️  Invalid MCP_SERVERS format. Expected 'name:url', got: %s", serverPair)
				continue
			}
			serverName := parts[0]
			serverType := parts[1] // "http" or "stdio"
			serverURL := parts[2]

			if err := srv.mcpServerManager.AddServer(ctx, serverName, serverType, serverURL); err != nil {
				log.Printf("❌ Failed to initialize MCP server %s: %v", serverName, err)
				// Don't fail orchestrator startup; just skip this server
				continue
			}
		}
	}

	// Method 2: From config file (JSON)
	mcpConfigPath := os.Getenv("MCP_CONFIG_PATH")
	if mcpConfigPath != "" {
		log.Printf("🔌 Loading MCP servers from config: %s", mcpConfigPath)
		if err := loadMCPServersFromConfig(ctx, mcpConfigPath, srv); err != nil {
			log.Printf("❌ Failed to load MCP config: %v", err)
		}
	}

	// Register all MCP server tools to the tool registry
	if err := srv.mcpServerManager.RegisterAllToRegistry(srv.toolRegistry); err != nil {
		log.Printf("❌ Failed to register MCP tools: %v", err)
	}
}

// loadMCPServersFromConfig loads MCP server configuration from a JSON file
func loadMCPServersFromConfig(ctx context.Context, configPath string, srv *server) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Updated to accurately extract the "type" field from your server config configuration layout
	var config struct {
		Servers []struct {
			Name string `json:"name"`
			Type string `json:"type"` // "http" or "stdio"
			URL  string `json:"url"`  // Endpoint address string or process execution path
		} `json:"servers"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	for _, serverCfg := range config.Servers {
		log.Printf("🔌 Adding MCP server from config: %s (%s via %s)", serverCfg.Name, serverCfg.URL, serverCfg.Type)

		// Adjust this line to match your manager method signature if you changed it,
		// or pass both parameters sequentially to handle stdio execution pipelines.
		if err := srv.mcpServerManager.AddServer(ctx, serverCfg.Name, serverCfg.Type, serverCfg.URL); err != nil {
			log.Printf("❌ Failed to initialize MCP server %s: %v", serverCfg.Name, err)
		}
	}

	return nil
}

// ============= STEP 4: Initialize tools method (if not already added) =============

func (s *server) initializeTools() {
	s.toolRegistry = s.workflow.InitializeToolRegistry()
	log.Println("🎯 Tool registry initialized and wired to orchestrator")
}

// ============= STEP 5: Update Chat method to use MCP tools =============

func (s *server) Chat(ctx context.Context, req *v1.ChatRequest) (*v1.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	// 1. Extract context
	lastMsg := req.GetMessages()[len(req.GetMessages())-1]
	userMsg := lastMsg.GetContent()

	userID := "default_user"

	isVision := len(lastMsg.GetImages()) > 0
	var imageData string
	if isVision {
		imageData = lastMsg.GetImages()[0]
	}

	// 2. Semantic Session Lookup
	sessionID, parentVectorID := s.lookupOrInitializeSession(ctx, userID, userMsg)

	// 3. ENHANCED: Call workflow WITH tool support (includes MCP tools)
	var result string
	var err error
	if s.toolRegistry != nil {
		result, err = s.workflow.RunWithTools(ctx, sessionID, userMsg, isVision, imageData, s.toolRegistry)
	} else {
		// Fallback to original run if tools not initialized
		result, err = s.workflow.Run(ctx, sessionID, userMsg, isVision, imageData)
	}

	if err != nil {
		log.Printf("Workflow error: %v", err)
		return nil, err
	}

	// 4. ASYNC PERSISTENCE: Save interaction + Graph Edge
	if s.qdrantClient != nil {
		go func(uID, sessID, pVectorID, promptText, respText string) {
			saveCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			log.Printf("DEBUG: Async interaction vector indexing initiated for session: %s", sessID)

			newVectorID, err := orchestrator.SaveInteractionToVectorDB(saveCtx, s.embeddingModel, s.qdrantClient, sessID, promptText, respText, pVectorID)
			if err != nil {
				log.Printf("ERROR: Failed to save interaction vector: %v", err)
				return
			}

			s.updateGraphContext(uID, sessID, newVectorID)

			log.Printf("✅ Success: Complete exchange for session %s indexed. Graph link updated.", sessID)
		}(userID, sessionID, parentVectorID, userMsg, result)
	}

	return &v1.ChatResponse{
		Message: &v1.Message{
			Role:    "assistant",
			Content: result,
		},
	}, nil
}

// ============= STEP 6: Add ListMCPTools RPC (OPTIONAL) =============

func (s *server) ListMCPTools(ctx context.Context, req *v1.ListMCPToolsRequest) (*v1.ListMCPToolsResponse, error) {
	allTools := s.mcpServerManager.ListAllTools()

	toolsByServer := make(map[string]*v1.ToolList)

	for serverName, toolDefs := range allTools {
		toolList := &v1.ToolList{}
		for _, toolDef := range toolDefs {
			schemaJSON, _ := json.Marshal(toolDef.InputSchema)
			tool := &v1.Tool{
				Name:            toolDef.Name,
				Description:     toolDef.Description,
				InputSchemaJson: string(schemaJSON),
			}
			toolList.Tools = append(toolList.Tools, tool)
		}
		toolsByServer[serverName] = toolList
	}

	return &v1.ListMCPToolsResponse{
		ToolsByServer: toolsByServer,
	}, nil
}

// ============= STEP 7: Configuration file example =============

/*
Create a file: mcp-servers-config.json

{
  "servers": [
    {
      "name": "gmail",
      "url": "http://localhost:3001/mcp"
    },
    {
      "name": "notion",
      "url": "http://localhost:3002/mcp"
    },
    {
      "name": "github",
      "url": "http://localhost:3003/mcp"
    }
  ]
}

Then run orchestrator with:
MCP_CONFIG_PATH=./mcp-servers-config.json go run main.go

OR using environment variables:
MCP_SERVERS="gmail:http://localhost:3001/mcp,notion:http://localhost:3002/mcp" go run main.go
*/
