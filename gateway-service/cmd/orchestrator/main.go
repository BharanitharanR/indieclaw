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
	graphMu        sync.RWMutex
	activeSessions map[string]*SessionNode
}

func (s *server) Chat(ctx context.Context, req *v1.ChatRequest) (*v1.ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	// 1. Extract context
	lastMsg := req.GetMessages()[len(req.GetMessages())-1]
	userMsg := lastMsg.GetContent()

	// Defaulting to "default_user" for now, ideally extract from auth header
	userID := "default_user"

	isVision := len(lastMsg.GetImages()) > 0
	var imageData string
	if isVision {
		imageData = lastMsg.GetImages()[0]
	}

	// 2. Semantic Session Lookup (Now using userID and returning graph edge)
	sessionID, parentVectorID := s.lookupOrInitializeSession(ctx, userID, userMsg)

	// 3. Call your workflow
	result, err := s.workflow.Run(ctx, sessionID, userMsg, isVision, imageData)
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

			// Save and get the NEW vector point ID
			newVectorID, err := orchestrator.SaveInteractionToVectorDB(saveCtx, s.embeddingModel, s.qdrantClient, sessID, promptText, respText, pVectorID)
			if err != nil {
				log.Printf("ERROR: Failed to save interaction vector: %v", err)
				return
			}

			// Update the Graph Memory for this user
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
		workflow:       wf,
		textModel:      textModel,
		visionModel:    visionModel,
		embeddingModel: embeddingModel,
		activeSessions: make(map[string]*SessionNode),
	}

	// Fire up Qdrant initialization concurrently in its own goroutine thread
	go func() {
		log.Println("🔄 Background thread: Initializing local Qdrant collection...")

		// Infinite retry strategy loop until local binary responds
		for {
			client, err := initdb.InitializeVectorStore()
			if err != nil {
				log.Printf("❌ Qdrant not ready yet (%v). Retrying in 3 seconds...", err)
				time.Sleep(3 * time.Second)
				continue
			}

			// Assign the live client instance to the running server memory
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

	log.Printf("🚀 Orchestrator running on :9000 [Text: %s, Vision: %s]", textModel, visionModel)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
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
