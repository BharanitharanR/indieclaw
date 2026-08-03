package orchestrator

import (
	"context"
	"log"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

type AgentHandler struct {
	callbacks.SimpleHandler
}

func NewAgentHandler() *AgentHandler {
	return &AgentHandler{}
}

func (h *AgentHandler) HandleLLMStart(ctx context.Context, prompts []string) {
	log.Printf("🔄 LLM START (prompts: %d)\n", len(prompts))
}

func (h *AgentHandler) HandleLLMGenerateContentStart(
	ctx context.Context,
	ms []llms.MessageContent,
) {
	// Suppress detailed content start logs
}

func (h *AgentHandler) HandleLLMGenerateContentEnd(
	ctx context.Context,
	res *llms.ContentResponse,
) {
	// Suppress detailed content end logs
}

func (h *AgentHandler) HandleLLMError(ctx context.Context, err error) {
	log.Printf("LLM ERROR: %v", err)
}

func (h *AgentHandler) HandleChainStart(
	ctx context.Context,
	inputs map[string]any,
) {
	log.Println("📋 Chain started")
}

func (h *AgentHandler) HandleChainEnd(
	ctx context.Context,
	outputs map[string]any,
) {
	log.Println("✅ Chain completed")
}

func (h *AgentHandler) HandleChainError(
	ctx context.Context,
	err error,
) {
	log.Printf("❌ Chain error: %v", err)
}

func (h *AgentHandler) HandleToolStart(
	ctx context.Context,
	input string,
) {
	log.Println("🔧 Tool executing...")
}

func (h *AgentHandler) HandleToolEnd(
	ctx context.Context,
	output string,
) {
	log.Println("✅ Tool completed")
}

func (h *AgentHandler) HandleToolError(
	ctx context.Context,
	err error,
) {
	log.Printf("❌ Tool error: %v", err)
}

func (h *AgentHandler) HandleAgentAction(
	ctx context.Context,
	action schema.AgentAction,
) {
	log.Printf("🎯 Agent action: %s", action.Tool)
}

func (h *AgentHandler) HandleAgentFinish(
	ctx context.Context,
	finish schema.AgentFinish,
) {
	log.Println("🏁 Agent finished")
}

func (h *AgentHandler) HandleRetrieverStart(
	ctx context.Context,
	query string,
) {
	log.Println("🔍 Retriever searching...")
}

func (h *AgentHandler) HandleRetrieverEnd(
	ctx context.Context,
	query string,
	documents []schema.Document,
) {
	log.Printf("✅ Retriever found %d documents", len(documents))
}

func (h *AgentHandler) HandleStreamingFunc(
	ctx context.Context,
	chunk []byte,
) {
	// Suppress token streaming logs to reduce noise
	// Tokens are logged in real-time and clutter the output
}

func (h *AgentHandler) HandleText(
	ctx context.Context,
	text string,
) {
	// Suppress intermediate text output
}
