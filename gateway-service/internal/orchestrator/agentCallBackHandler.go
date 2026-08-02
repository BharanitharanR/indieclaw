package orchestrator

import (
	"context"
	"encoding/json"
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
	log.Println("========== LLM START ==========")

	for i, p := range prompts {
		log.Printf("Prompt %d:\n%s\n", i+1, p)
	}
}

func (h *AgentHandler) HandleLLMGenerateContentStart(
	ctx context.Context,
	ms []llms.MessageContent,
) {
	log.Println("LLM GenerateContent START")
}

func (h *AgentHandler) HandleLLMGenerateContentEnd(
	ctx context.Context,
	res *llms.ContentResponse,
) {
	log.Println("LLM GenerateContent END")

	b, _ := json.MarshalIndent(res, "", "  ")
	log.Println(string(b))
}

func (h *AgentHandler) HandleLLMError(ctx context.Context, err error) {
	log.Printf("LLM ERROR: %v", err)
}

func (h *AgentHandler) HandleChainStart(
	ctx context.Context,
	inputs map[string]any,
) {
	log.Println("========== CHAIN START ==========")

	b, _ := json.MarshalIndent(inputs, "", "  ")
	log.Println(string(b))
}

func (h *AgentHandler) HandleChainEnd(
	ctx context.Context,
	outputs map[string]any,
) {
	log.Println("========== CHAIN END ==========")

	b, _ := json.MarshalIndent(outputs, "", "  ")
	log.Println(string(b))
}

func (h *AgentHandler) HandleChainError(
	ctx context.Context,
	err error,
) {
	log.Printf("CHAIN ERROR: %v", err)
}

func (h *AgentHandler) HandleToolStart(
	ctx context.Context,
	input string,
) {
	log.Println("========== TOOL START ==========")
	log.Println(input)
}

func (h *AgentHandler) HandleToolEnd(
	ctx context.Context,
	output string,
) {
	log.Println("========== TOOL END ==========")
	log.Println(output)
}

func (h *AgentHandler) HandleToolError(
	ctx context.Context,
	err error,
) {
	log.Printf("TOOL ERROR: %v", err)
}

func (h *AgentHandler) HandleAgentAction(
	ctx context.Context,
	action schema.AgentAction,
) {
	log.Println("========== AGENT ACTION ==========")

	log.Printf("Tool       : %s", action.Tool)
	log.Printf("Tool Input : %s", action.ToolInput)
	log.Printf("Log:\n%s", action.Log)
}

func (h *AgentHandler) HandleAgentFinish(
	ctx context.Context,
	finish schema.AgentFinish,
) {
	log.Println("========== AGENT FINISH ==========")

	b, _ := json.MarshalIndent(finish.ReturnValues, "", "  ")
	log.Println(string(b))
}

func (h *AgentHandler) HandleRetrieverStart(
	ctx context.Context,
	query string,
) {
	log.Printf("Retriever query: %s", query)
}

func (h *AgentHandler) HandleRetrieverEnd(
	ctx context.Context,
	query string,
	documents []schema.Document,
) {
	log.Printf("Retriever returned %d documents", len(documents))
}

func (h *AgentHandler) HandleStreamingFunc(
	ctx context.Context,
	chunk []byte,
) {
	log.Print(string(chunk))
}

func (h *AgentHandler) HandleText(
	ctx context.Context,
	text string,
) {
	log.Println(text)
}
