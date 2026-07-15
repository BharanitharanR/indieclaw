package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gateway-service/internal/chat"
	"gateway-service/internal/mcp"
	"io"
	"log"
	"net/http"
	"time"
)

type OllamaService struct {
	httpClient   *http.Client
	ollamaBase   string
	defaultModel string
	mcpRegistry  *mcp.ToolRegistry
	queuePermit  chan struct{} // Semaphore to prevent Ollama GPU memory exhaustion
}

func NewOllamaService(ollamaBase string, defaultModel string, registry *mcp.ToolRegistry) *OllamaService {
	sem := make(chan struct{}, 1) // Capacity = 1 permit
	sem <- struct{}{}             // Initial permit token

	return &OllamaService{
		httpClient:   &http.Client{Timeout: 180 * time.Second}, // Extended timeout for Vision processing
		ollamaBase:   ollamaBase,
		defaultModel: defaultModel,
		mcpRegistry:  registry,
		queuePermit:  sem,
	}
}

// RegisterRoutes registers the REST endpoint expected by your WhatsApp client
func (s *OllamaService) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/chat", s.handleProxyChat)
	mux.HandleFunc("POST /api/v1/generate", s.handleProxyGenerate)
}

func (s *OllamaService) handleProxyChat(w http.ResponseWriter, r *http.Request) {
	var req chat.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request JSON", http.StatusBadRequest)
		return
	}

	// Default model fallback if not provided by client
	if req.Model == "" {
		req.Model = s.defaultModel
	}

	// Ensure non-streaming response for simpler JS client consumption
	req.Stream = false

	log.Printf("[Chat Request] Model: %s | Messages: %d", req.Model, len(req.Messages))

	resp, err := s.AskOllama(r.Context(), &req)
	if err != nil {
		log.Printf("[Error] Ollama request failed: %v", err)
		http.Error(w, fmt.Sprintf("Bad Gateway: %v", err), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *OllamaService) handleProxyGenerate(w http.ResponseWriter, r *http.Request) {
	s.proxyStream(w, r, fmt.Sprintf("%s/api/generate", s.ollamaBase))
}

func (s *OllamaService) AskOllama(ctx context.Context, req *chat.ChatRequest) (*chat.ChatResponse, error) {
	// 1. Inject System Prompt (using chat.Message)

	// 2. Filter Tools
	allTools := s.mcpRegistry.GetAllTools()
	var ollamaTools []chat.Tool
	for _, t := range allTools {
		if t.Name == "search" || t.Name == "fetch_content" {
			ollamaTools = append(ollamaTools, chat.Tool{
				Type: "function",
				Function: chat.ToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			})
		}
	}
	req.Tools = ollamaTools

	url := fmt.Sprintf("%s/api/chat", s.ollamaBase)

	for {
		jsonBytes, _ := json.Marshal(req)
		httpReq, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(httpReq)
		if err != nil {
			return nil, err
		}

		var chatResp chat.ChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()

		if len(chatResp.Message.ToolCalls) > 0 {
			req.Messages = append(req.Messages, chatResp.Message)

			for _, tc := range chatResp.Message.ToolCalls {
				var args map[string]any
				println("Tool Call:", tc.Function.Name, "Args:", tc.Function.Arguments)
				json.Unmarshal([]byte(tc.Function.Arguments), &args)

				toolResult, _ := s.mcpRegistry.Execute(ctx, tc.Function.Name, args)

				// Use chat.Message here instead of chat.ChatMessage
				req.Messages = append(req.Messages, chat.ChatMessage{
					Role:    "tool",
					Content: toolResult,
				})
			}
			continue
		}
		return &chatResp, nil
	}
}

// Helper: Converts your MCP tool definitions to Ollama's expected JSON format

func (s *OllamaService) proxyStream(w http.ResponseWriter, r *http.Request, targetURL string) {
	// Acquire permit lock
	<-s.queuePermit
	defer func() { s.queuePermit <- struct{}{} }() // Release lock

	proxyReq, err := http.NewRequestWithContext(r.Context(), "POST", targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to build proxy request", http.StatusInternalServerError)
		return
	}

	if cType := r.Header.Get("Content-Type"); cType != "" {
		proxyReq.Header.Set("Content-Type", cType)
	}

	resp, err := s.httpClient.Do(proxyReq)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	_, _ = io.Copy(w, resp.Body)
}
