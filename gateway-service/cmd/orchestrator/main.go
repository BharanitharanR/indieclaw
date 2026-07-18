package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	// Added for environment variables
	"gateway-service/internal/orchestrator"
	v1 "gateway-service/proto/gateway/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	v1.UnimplementedGatewayServiceServer
	workflow    *orchestrator.AgentWorkflow
	textModel   string // Store these
	visionModel string // Store these
}

func (s *server) Chat(ctx context.Context, req *v1.ChatRequest) (*v1.ChatResponse, error) {
	var imageData string
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	// 1. Get models from environment
	textModel := os.Getenv("TEXT_MODEL")
	visionModel := os.Getenv("VISION_MODEL")

	// 2. Extract the actual user message (last message)
	lastMsg := req.GetMessages()[len(req.GetMessages())-1]
	userMsg := lastMsg.GetContent()

	// 3. Determine if this is a vision request and select the model
	isVision := len(lastMsg.GetImages()) > 0
	log.Default().Printf("DEBUG: Received message. IsVision: %v, UserMsg: %s, ImagesCount: %d", isVision, userMsg, len(lastMsg.GetImages()))
	selectedModel := textModel
	if isVision {
		selectedModel = visionModel
		imageData = lastMsg.GetImages()[0]
		log.Printf("DEBUG: Processing [Vision: %v] with model: %s, Image Data: %d ", isVision, selectedModel, len(imageData))
	}

	log.Printf("DEBUG: Processing [Vision: %v] with model: %s, Input: %s ", isVision, selectedModel, userMsg)

	// 4. Call your workflow
	// Note: You must pass 'userMsg' and 'sessionID' to the workflow
	// so the LLM actually receives the text.
	result, err := s.workflow.Run(ctx, "default-session", userMsg, isVision, imageData)
	if err != nil {
		log.Printf("Workflow error: %v", err)
		return nil, err
	}

	return &v1.ChatResponse{
		Message: &v1.Message{
			Role:    "assistant",
			Content: result,
		},
	}, nil
}
func main() {
	// 1. Load configuration from Environment Variables
	textModel := os.Getenv("TEXT_MODEL")
	if textModel == "" {
		textModel = "qwen3:8b" // Default fallback
	}
	visionModel := os.Getenv("VISION_MODEL")
	if visionModel == "" {
		visionModel = "gemma4:e2b" // Default fallback
	}

	// Use background context for initialization
	ctx := context.Background()
	wf, err := orchestrator.NewAgentWorkflow(ctx, textModel, visionModel)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}

	// 3. Create the gRPC server
	s := grpc.NewServer()

	// 4. Register the server implementation
	// We inject the models and workflow here once at startup
	srv := &server{
		workflow:    wf,
		textModel:   textModel,
		visionModel: visionModel,
	}
	v1.RegisterGatewayServiceServer(s, srv)

	// Enable reflection for easy testing
	reflection.Register(s)

	// 5. Start listening on TCP
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to listen on :9000: %v", err)
	}

	log.Printf("🚀 Orchestrator running on :9000 [Text: %s, Vision: %s]", textModel, visionModel)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
