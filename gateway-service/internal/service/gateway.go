package service

import (
	"context"
	"fmt"
	"log"

	"gateway-service/internal/config"
	pb "gateway-service/proto/gateway/v1"
)

// GatewayServer implements the gRPC GatewayServiceServer interface
type GatewayServer struct {
	pb.UnimplementedGatewayServiceServer
	cfg *config.Config
}

func NewGatewayServer(cfg *config.Config) *GatewayServer {
	return &GatewayServer{
		cfg: cfg,
	}
}

// Chat handles incoming user prompts from clients (e.g. WhatsApp adapter)
func (s *GatewayServer) Chat(ctx context.Context, req *pb.ChatRequest) (*pb.ChatResponse, error) {
	if len(req.GetMessages()) == 0 {
		return nil, fmt.Errorf("request contains no messages")
	}

	sessionID := req.GetSessionId()
	lastMessage := req.GetMessages()[len(req.GetMessages())-1]

	log.Printf("[gRPC Chat] Session: %s | Role: %s | Prompt: %s",
		sessionID, lastMessage.GetRole(), lastMessage.GetContent())

	// TODO: Step 3 will connect this to the Tool Planner & Primary LLM
	// For now, return a mock response to confirm gRPC server is functioning
	mockReply := fmt.Sprintf("Echo from Go Gateway! Session ID: %s. You said: %s", sessionID, lastMessage.GetContent())

	return &pb.ChatResponse{
		Reply:     mockReply,
		UsedTools: []string{}, // Empty for now
	}, nil
}
