package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	initdb "gateway-service/cmd/init_db"
	"gateway-service/internal/orchestrator"
	v1 "gateway-service/proto/gateway/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	v1.UnimplementedGatewayServiceServer
	workflow       *orchestrator.AgentWorkflow
	qdrantClient   *qdrant.Client
	textModel      string
	visionModel    string
	embeddingModel string
	graphMu        sync.RWMutex
	activeSessions map[string]*SessionNode
	toolRegistry   *orchestrator.ToolRegistry
}

type SessionNode struct {
	SessionID        string
	PreviousVectorID string
	LastSeen         time.Time
}

func main() {
	initLogger()

	// Load persona configuration first
	personaName := os.Getenv("PERSONA_NAME")
	if personaName == "" {
		personaName = "default"
	}

	persona, err := orchestrator.LoadPersona(personaName)
	if err != nil {
		log.Fatalf("Failed to load persona %q: %v", personaName, err)
	}

	// Use persona's model settings if provided, otherwise use environment variables
	textModel := os.Getenv("TEXT_MODEL")
	if textModel == "" {
		textModel = persona.TextModel
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel == "" {
		visionModel = persona.VisionModel
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
		workflow:       wf,
		textModel:      textModel,
		visionModel:    visionModel,
		embeddingModel: embeddingModel,
		activeSessions: make(map[string]*SessionNode),
	}

	// 🎯 Initialize tool registry once
	srv.initializeTools()

	// 🌐 Initialize MCP servers and load adapter tools via config path
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

	log.Printf("🚀 Orchestrator running on :9000 [Persona: %s, Text: %s, Vision: %s, Tools: ENABLED]", persona.Name, textModel, visionModel)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
func initLogger() {
	logFilePath := os.Getenv("LOG_PATH")
	file, err := os.OpenFile(
		logFilePath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}

	log.SetOutput(file)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)
}

// ============= MCP INITIALIZATION =============

func initializeMCPServers(ctx context.Context, srv *server) {
	mcpConfigPath := os.Getenv("MCP_CONFIG_PATH")
	if mcpConfigPath == "" {
		log.Println("ℹ️ No MCP_CONFIG_PATH provided. Skipping MCP server initialization.")
		return
	}

	log.Printf("🔌 Loading MCP servers from config path: %s", mcpConfigPath)

	mcpTools, err := orchestrator.LoadMCPToolsFromConfig(ctx, mcpConfigPath)
	if err != nil {
		log.Printf("❌ Failed to load MCP tools from config: %v", err)
		return
	}

	// Register each wrapped MCP tool directly into the server's ToolRegistry
	if srv.toolRegistry != nil {
		for _, tool := range mcpTools {
			// If your ToolRegistry accepts standard tools.Tool or custom definitions,
			// register them directly so GetLangChainTools() picks them up.
			_ = srv.toolRegistry.RegisterLangChainTool(tool)
		}
	}

	log.Printf("✅ Successfully loaded, wrapped, and registered %d MCP tools into registry.", len(mcpTools))
}

// ============= TOOL REGISTRY SETUP =============

func (s *server) initializeTools() {
	s.toolRegistry = s.workflow.InitializeToolRegistry()
	log.Println("🎯 Tool registry initialized and wired to orchestrator")
}

// ============= SESSION & CHAT HANDLERS =============

func (s *server) lookupOrInitializeSession(ctx context.Context, userID, query string) (string, string) {
	s.graphMu.RLock()
	activeNode, exists := s.activeSessions[userID]
	s.graphMu.RUnlock()

	if exists && time.Since(activeNode.LastSeen) < 5*time.Minute {
		log.Printf("🔗 Graph Context Link: User %s active within 5m. Extending session: %s", userID, activeNode.SessionID)
		return activeNode.SessionID, activeNode.PreviousVectorID
	}

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
				if topMatch.Score > 0.80 {
					payload := topMatch.Payload
					sessID := payload["session_id"].GetStringValue()

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

	newSessionID := uuid.NewString()
	log.Printf("ℹ️ Initializing completely fresh session track: %s", newSessionID)
	return newSessionID, ""
}

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

func (s *server) Chat(ctx context.Context, req *v1.ChatRequest) (*v1.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	lastMsg := req.GetMessages()[len(req.GetMessages())-1]
	userMsg := lastMsg.GetContent()

	userID := "default_user"

	isVision := len(lastMsg.GetImages()) > 0
	var imageData string
	if isVision {
		imageData = lastMsg.GetImages()[0]
	}

	sessionID, parentVectorID := s.lookupOrInitializeSession(ctx, userID, userMsg)

	var result string
	var err error
	if s.toolRegistry != nil {
		result, err = s.workflow.RunWithTools(ctx, sessionID, userMsg, isVision, imageData, s.toolRegistry)
	} else {
		result, err = s.workflow.Run(ctx, sessionID, userMsg, isVision, imageData)
	}

	if err != nil {
		log.Printf("Workflow error: %v", err)
		return nil, err
	}

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
	log.Printf("result %s", result)

	return &v1.ChatResponse{
		Message: &v1.Message{
			Role:    "assistant",
			Content: result,
		},
	}, nil
}

func (s *server) ListMCPTools(ctx context.Context, req *v1.ListMCPToolsRequest) (*v1.ListMCPToolsResponse, error) {
	mcpConfigPath := os.Getenv("MCP_CONFIG_PATH")
	if mcpConfigPath == "" {
		return &v1.ListMCPToolsResponse{ToolsByServer: make(map[string]*v1.ToolList)}, nil
	}

	_, _ = orchestrator.LoadMCPToolsFromConfig(ctx, mcpConfigPath)

	return &v1.ListMCPToolsResponse{
		ToolsByServer: make(map[string]*v1.ToolList),
	}, nil
}
