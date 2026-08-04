package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	amqp "github.com/rabbitmq/amqp091-go"

	initdb "gateway-service/cmd/init_db"
	"gateway-service/internal/orchestrator"
	v1 "gateway-service/proto/gateway/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	v1.UnimplementedGatewayServiceServer
	workflow       *orchestrator.AgentWorkflow
	pipeline       *orchestrator.Pipeline
	persona        *orchestrator.PersonaDefinition
	personaExecutor orchestrator.PersonaExecutor
	qdrantClient   *qdrant.Client
	textModel      string
	visionModel    string
	embeddingModel string
	graphMu        sync.RWMutex
	activeSessions map[string]*SessionNode
	toolRegistry   *orchestrator.ToolRegistry
	rabbitmqConn   *amqp.Connection
	rabbitmqCh     *amqp.Channel
	reqQueue       amqp.Queue
	respQueue      amqp.Queue
}

type SessionNode struct {
	SessionID        string
	PreviousVectorID string
	LastSeen         time.Time
}

func main() {
	initLogger()

	// ============= LOAD PERSONA =============
	personaName := os.Getenv("PERSONA_NAME")
	if personaName == "" {
		personaName = "generic_assistant"
	}

	personaDef, err := orchestrator.LoadPersona(personaName)
	if err != nil {
		log.Printf("⚠️  Failed to load persona %q, attempting fallback to generic_assistant: %v", personaName, err)
		personaDef, err = orchestrator.LoadPersona("generic_assistant")
		if err != nil {
			log.Fatalf("❌ Failed to load fallback persona: %v", err)
		}
	}

	log.Printf("✅ Loaded persona: %s (v%d)", personaDef.Name, personaDef.Version)

	// Use persona's model settings if provided, otherwise use environment variables
	textModel := os.Getenv("TEXT_MODEL")
	if textModel == "" {
		textModel = personaDef.TextModel
	}
	if textModel == "" {
		textModel = "qwen2:7b"
	}

	visionModel := os.Getenv("VISION_MODEL")
	if visionModel == "" {
		visionModel = personaDef.VisionModel
	}
	if visionModel == "" {
		visionModel = "llava:7b"
	}

	embeddingModel := os.Getenv("EMBEDDING_MODEL")
	if embeddingModel == "" {
		embeddingModel = "nomic-embed-text"
	}

	// ============= INITIALIZE LLMS & PIPELINE =============
	ctx := context.Background()
	wf, err := orchestrator.NewAgentWorkflow(ctx, textModel, visionModel)
	if err != nil {
		log.Fatalf("❌ Failed to create workflow: %v", err)
	}

	// Initialize Pipeline with the LLMs from workflow
	pipeline := orchestrator.NewPipeline(
		nil, // Will be set when Qdrant is ready
		wf.TextAgent,
		wf.TextAgent,
		wf.VisionAgent,
		embeddingModel,
	)

	// Create persona executor for intent classification and response validation
	personaExecutor := orchestrator.NewPersonaExecutor(wf.TextAgent)

	s := grpc.NewServer()

	srv := &server{
		workflow:        wf,
		pipeline:        pipeline,
		persona:         personaDef,
		personaExecutor: personaExecutor,
		textModel:       textModel,
		visionModel:     visionModel,
		embeddingModel:  embeddingModel,
		activeSessions:  make(map[string]*SessionNode),
	}

	// 🎯 Initialize tool registry once
	srv.initializeTools()

	// 🌐 Initialize MCP servers and load adapter tools via config path
	initializeMCPServers(ctx, srv)

	// 🐰 Initialize RabbitMQ queue consumer
	if err := srv.initializeRabbitMQ(); err != nil {
		log.Printf("⚠️  RabbitMQ initialization failed: %v (gRPC-only mode)", err)
	} else {
		log.Println("✅ RabbitMQ queue consumer initialized")
		// Start consuming messages from orchestrator.requests queue
		go srv.consumeRequestQueue()
	}

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

			// Reinitialize pipeline with Qdrant client now that it's ready
			srv.pipeline = orchestrator.NewPipeline(
				client,
				wf.TextAgent,
				wf.TextAgent,
				wf.VisionAgent,
				embeddingModel,
			)

			log.Println("✅ Background thread: Qdrant client connected and successfully wired.")
			log.Println("✅ Background thread: Pipeline reinitialized with Qdrant client.")
			break
		}
	}()

	v1.RegisterGatewayServiceServer(s, srv)
	reflection.Register(s)

	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on :9000: %v", err)
	}

	log.Printf("🚀 Orchestrator running on :9000")
	log.Printf("   Persona: %s (v%d)", personaDef.Name, personaDef.Version)
	log.Printf("   Text Model: %s | Vision Model: %s", textModel, visionModel)
	log.Printf("   Embedding: %s | Queue: RabbitMQ", embeddingModel)
	log.Printf("   Intents: %d | Validation Gates: %d", len(personaDef.Intents), len(personaDef.ValidationGates))

	// Handle graceful shutdown
	defer func() {
		if srv.rabbitmqCh != nil {
			srv.rabbitmqCh.Close()
		}
		if srv.rabbitmqConn != nil {
			srv.rabbitmqConn.Close()
		}
		s.GracefulStop()
		log.Println("✅ Orchestrator shutdown complete")
	}()

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

	log.Printf("\n📨 CHAT REQUEST [Persona: %s]", s.persona.Name)
	log.Printf("   User: %s | Session: %s | Vision: %v", userID, sessionID, isVision)

	// Classify intent using persona executor
	pctx := &orchestrator.PersonaContext{
		Definition: s.persona,
		Evidence:   map[string]interface{}{"message": userMsg},
	}

	intent, confidence := s.personaExecutor.ClassifyIntent(userMsg, pctx)
	log.Printf("   Intent: %s (confidence: %.0f%%)", intent, confidence*100)

	// Get response mode for this intent
	rule := s.personaExecutor.GetModeForIntent(intent, s.persona)
	log.Printf("   Mode: %s | Search: %v | Template: %s", rule.Mode, rule.SearchEnabled, rule.Template)

	var result string
	var err error
	if s.toolRegistry != nil {
		result, err = s.workflow.RunWithTools(ctx, sessionID, userMsg, isVision, imageData, s.toolRegistry)
	} else {
		result, err = s.workflow.Run(ctx, sessionID, userMsg, isVision, imageData)
	}

	if err != nil {
		log.Printf("❌ Workflow error: %v", err)
		return nil, err
	}

	// Validate response against persona rules
	issues := s.personaExecutor.ValidateResponseAgainstRules(result, rule.Mode, s.persona)
	if len(issues) > 0 {
		log.Printf("⚠️  Response validation issues (%d):", len(issues))
		for _, issue := range issues {
			log.Printf("   - %s [%s]: %s", issue.Gate, issue.Severity, issue.Message)
			if issue.Severity == "error" {
				log.Printf("     Fix: %s", issue.Fix)
			}
		}
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
			PersonaMetadata: &v1.PersonaMetadata{
				PersonaName: s.persona.Name,
				Intent:      intent,
				Mode:        rule.Mode,
				Confidence:  confidence,
			},
		},
	}, nil
}

// GetPersona returns details about the current active persona
func (s *server) GetPersona(ctx context.Context, req *v1.GetPersonaRequest) (*v1.PersonaResponse, error) {
	if s.persona == nil {
		return nil, fmt.Errorf("no persona loaded")
	}

	intentList := make([]*v1.IntentConfig, 0)
	for name, rule := range s.persona.Intents {
		intentList = append(intentList, &v1.IntentConfig{
			Name:     name,
			Mode:     rule.Mode,
			Template: rule.Template,
		})
	}

	canHandle := make([]string, len(s.persona.Capabilities.CanHandle))
	copy(canHandle, s.persona.Capabilities.CanHandle)

	cannotHandle := make([]string, len(s.persona.Capabilities.CannotHandle))
	copy(cannotHandle, s.persona.Capabilities.CannotHandle)

	return &v1.PersonaResponse{
		Name:           s.persona.Name,
		Version:        int32(s.persona.Version),
		Description:    s.persona.Description,
		CanHandle:      canHandle,
		CannotHandle:   cannotHandle,
		Intents:        intentList,
		TextModel:      s.persona.TextModel,
		VisionModel:    s.persona.VisionModel,
	}, nil
}

// ListPersonas returns available personas
func (s *server) ListPersonas(ctx context.Context, req *v1.ListPersonasRequest) (*v1.ListPersonasResponse, error) {
	personas, err := orchestrator.ListAvailablePersonas()
	if err != nil {
		log.Printf("❌ Failed to list personas: %v", err)
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}

	var personalInfos []*v1.PersonaInfo
	for _, name := range personas {
		persona, err := orchestrator.LoadPersona(name)
		if err != nil {
			log.Printf("⚠️  Failed to load persona %q: %v", name, err)
			continue
		}

		isActive := (name == s.persona.Name)
		personalInfos = append(personalInfos, &v1.PersonaInfo{
			Name:        persona.Name,
			Version:     int32(persona.Version),
			Description: persona.Description,
			IsActive:    isActive,
		})
	}

	return &v1.ListPersonasResponse{
		Personas: personalInfos,
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

// ============= RABBITMQ INITIALIZATION =============

func (s *server) initializeRabbitMQ() error {
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://indieclaw:secretpass@localhost:5672/"
	}

	conn, err := amqp.Dial(rabbitmqURL)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare request queue (durable, FIFO)
	reqQueue, err := ch.QueueDeclare(
		"orchestrator.requests",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare request queue: %w", err)
	}

	// Declare response queue (durable, auto-expire after 1 hour)
	respQueue, err := ch.QueueDeclare(
		"orchestrator.responses",
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{"x-expires": 3600000}, // 1 hour TTL in milliseconds
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to declare response queue: %w", err)
	}

	// Set QoS to 1: process one message at a time per orchestrator instance
	err = ch.Qos(1, 0, false)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	s.rabbitmqConn = conn
	s.rabbitmqCh = ch
	s.reqQueue = reqQueue
	s.respQueue = respQueue

	log.Printf("🐰 RabbitMQ connected: requests=[%d], responses=[%d]", reqQueue.Messages, respQueue.Messages)
	return nil
}

// ============= RABBITMQ MESSAGE CONSUMER =============

type QueueRequest struct {
	CorrelationID  string   `json:"correlationId"`
	PhoneNumber    string   `json:"phoneNumber"`
	Message        string   `json:"message"`
	Timestamp      int64    `json:"timestamp"`
	IdempotencyKey string   `json:"idempotencyKey"`
	Images         []string `json:"images,omitempty"`
}

type QueueResponse struct {
	CorrelationID   string `json:"correlationId"`
	PhoneNumber     string `json:"phoneNumber"`
	Result          string `json:"result,omitempty"`
	Error           string `json:"error,omitempty"`
	Status          string `json:"status"` // "success" or "error"
	Timestamp       int64  `json:"timestamp"`
	ProcessingTimeMs int64 `json:"processingTimeMs"`
}

func (s *server) consumeRequestQueue() {
	if s.rabbitmqCh == nil {
		log.Println("⚠️  RabbitMQ channel not initialized, skipping consumer")
		return
	}

	msgs, err := s.rabbitmqCh.Consume(
		s.reqQueue.Name,
		"",    // consumer tag
		false, // auto-ack (we'll ack manually)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		log.Printf("❌ Failed to start consuming: %v", err)
		return
	}

	log.Printf("🎧 Started consuming from queue: %s", s.reqQueue.Name)

	for delivery := range msgs {
		var req QueueRequest
		err := json.Unmarshal(delivery.Body, &req)
		if err != nil {
			log.Printf("❌ Failed to unmarshal request: %v", err)
			delivery.Nack(false, false) // reject and don't requeue
			continue
		}

		log.Printf("📨 Processing request [%s] from %s", req.CorrelationID, req.PhoneNumber)

		// Process the request through the pipeline
		s.handleQueueRequest(req, delivery)
	}
}

func (s *server) handleQueueRequest(req QueueRequest, delivery amqp.Delivery) {
	startTime := time.Now()

	// Get the loaded persona
	persona := orchestrator.GetPersona()
	if persona == nil {
		log.Printf("❌ No persona loaded for request [%s]", req.CorrelationID)
		s.publishErrorResponse(req, "No persona configured", startTime)
		delivery.Ack(false)
		return
	}

	// Determine if this is a vision request
	isVision := len(req.Images) > 0
	var imageBase64 string
	if isVision && len(req.Images) > 0 {
		imageBase64 = req.Images[0]
	}

	// Execute pipeline with longer timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := s.pipeline.Execute(ctx, req.Message, req.CorrelationID, persona, isVision, imageBase64)

	if err != nil {
		log.Printf("❌ Pipeline error [%s]: %v", req.CorrelationID, err)
		s.publishErrorResponse(req, err.Error(), startTime)
	} else {
		log.Printf("✅ Pipeline completed [%s]: %d chars", req.CorrelationID, len(result))
		s.publishSuccessResponse(req, result, startTime)
	}

	// Acknowledge the message only after we've published the response
	delivery.Ack(false)
	log.Printf("✓ ACK'd request [%s]", req.CorrelationID)
}

func (s *server) publishSuccessResponse(req QueueRequest, result string, startTime time.Time) {
	resp := QueueResponse{
		CorrelationID:   req.CorrelationID,
		PhoneNumber:     req.PhoneNumber,
		Result:          result,
		Status:          "success",
		Timestamp:       time.Now().Unix(),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}

	s.publishResponse(resp)
}

func (s *server) publishErrorResponse(req QueueRequest, errMsg string, startTime time.Time) {
	resp := QueueResponse{
		CorrelationID:   req.CorrelationID,
		PhoneNumber:     req.PhoneNumber,
		Error:           errMsg,
		Status:          "error",
		Timestamp:       time.Now().Unix(),
		ProcessingTimeMs: time.Since(startTime).Milliseconds(),
	}

	s.publishResponse(resp)
}

func (s *server) publishResponse(resp QueueResponse) {
	if s.rabbitmqCh == nil {
		log.Printf("❌ RabbitMQ channel not available, cannot publish response [%s]", resp.CorrelationID)
		return
	}

	body, err := json.Marshal(resp)
	if err != nil {
		log.Printf("❌ Failed to marshal response: %v", err)
		return
	}

	err = s.rabbitmqCh.Publish(
		"",              // exchange
		s.respQueue.Name, // routing key (queue name)
		false,           // mandatory
		false,           // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: resp.CorrelationID,
			Body:          body,
		},
	)

	if err != nil {
		log.Printf("❌ Failed to publish response [%s]: %v", resp.CorrelationID, err)
		return
	}

	log.Printf("📤 Published response [%s] to %s", resp.CorrelationID, s.respQueue.Name)
}
