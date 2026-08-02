package orchestrator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"

	"github.com/ollama/ollama/api"
	"github.com/qdrant/go-client/qdrant"
)

type AgentWorkflow struct {
	textAgent   *ollama.LLM
	visionAgent *ollama.LLM
}

// ReActStep represents a parsed step from our internal execution planner
type ReActStep struct {
	ID             int    `json:"id"`
	Action         string `json:"action"`
	NeedsUserInput bool   `json:"needs_user_input"`
}

type ReActPlan struct {
	Steps []ReActStep `json:"steps"`
}

func NewAgentWorkflowCustom(ctx context.Context, textModelName, visionModelName string) (*AgentWorkflow, error) {
	// Fallback custom constructor if needed, mapping back to native Ollama LLM types for consistency
	textLLM, err := ollama.New(ollama.WithModel(textModelName), ollama.WithServerURL("http://localhost:11434"))
	if err != nil {
		return nil, err
	}

	visionLLM, err := ollama.New(ollama.WithModel(visionModelName), ollama.WithServerURL("http://localhost:11434"))
	if err != nil {
		return nil, err
	}

	return &AgentWorkflow{textAgent: textLLM, visionAgent: visionLLM}, nil
}

func NewAgentWorkflow(ctx context.Context, textModelName, visionModelName string) (*AgentWorkflow, error) {
	// Initialize native LangChain Ollama text client
	textLLM, err := ollama.New(
		ollama.WithModel(textModelName),
		ollama.WithServerURL("http://localhost:11434"),
	)
	if err != nil {
		return nil, err
	}

	// Initialize native LangChain Ollama vision client
	visionLLM, err := ollama.New(
		ollama.WithModel(visionModelName),
		ollama.WithServerURL("http://localhost:11434"),
	)
	if err != nil {
		return nil, err
	}

	return &AgentWorkflow{
		textAgent:   textLLM,
		visionAgent: visionLLM,
	}, nil
}

// Run executes the core ReAct planning loop using native LangChain model calls.
func (w *AgentWorkflow) Run(ctx context.Context, sessionID string, input string, isVision bool, imageBase64 string) (string, error) {
	if isVision {
		// 1. Decode the Base64 string into RAW BYTES
		imgBytes, err := base64.StdEncoding.DecodeString(imageBase64)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %v", err)
		}
		mimeType := http.DetectContentType(imgBytes)
		log.Printf("DEBUG: Detected MIME for file: %s (MIME: %s)", mimeType, mimeType)

		// 2. Construct LangChain multimodal content request
		contentParts := []llms.ContentPart{
			llms.TextPart(input),
			llms.ImageURLPart("data:image/jpeg;base64," + imageBase64),
		}

		resp, err := w.visionAgent.GenerateContent(ctx, []llms.MessageContent{
			{
				Role:  llms.ChatMessageTypeHuman,
				Parts: contentParts,
			},
		})
		if err != nil {
			return "", fmt.Errorf("vision model generation failed: %w", err)
		}

		if len(resp.Choices) > 0 {
			return resp.Choices[0].Content, nil
		}
		return "", fmt.Errorf("empty vision response")
	}

	// --- TEXT AGENT: CORE REACT STATE GRAPH LOOP ---

	planPrompt := fmt.Sprintf(`You are an intelligent Assistant Planner. 
Your goal is to answer the user or perform the task. 

- If the user's request is a clear question or command (like "What is free will?"), provide a step to "Answer directly" with "needs_user_input": false.
- ONLY emit a step with "needs_user_input": true if it is IMPOSSIBLE to proceed without specific user-provided credentials, file paths, or highly ambiguous instructions (like "Do it now" without context).
- Do not ask for clarification for philosophical, academic, or general knowledge topics.

Output your execution path STRICTLY as a valid JSON block.
JSON Structure:
{
  "steps": [
    {"id": 1, "action": "A brief description of your task", "needs_user_input": false}
  ]
}

User Prompt: "%s"`, input)

	rawPlanResp, err := llms.GenerateFromSinglePrompt(ctx, w.textAgent, planPrompt)
	if err != nil {
		return "", fmt.Errorf("planning phase generation failed: %w", err)
	}

	planString := extractString(rawPlanResp)

	var plan ReActPlan
	if err := json.Unmarshal([]byte(planString), &plan); err != nil {
		log.Printf("WARN: Failed parsing model plan JSON. Fallback to direct resolution. Error: %v Raw: %s", err, planString)
		directResp, err := llms.GenerateFromSinglePrompt(ctx, w.textAgent, input)
		return extractString(directResp), err
	}

	var observations []string

	// Step 2: Iterate through steps sequentially
	for _, step := range plan.Steps {
		log.Printf("⚙️ ReAct Executor processing Step [%d]: %s (Needs Input: %t)", step.ID, step.Action, step.NeedsUserInput)

		if step.NeedsUserInput {
			return fmt.Sprintf("⏸️ Paused for clarification: %s", step.Action), nil
		}
		executionTask := fmt.Sprintf("Task: %s. Provide the result or answer for this task.", step.Action)

		actionResult, err := llms.GenerateFromSinglePrompt(ctx, w.textAgent, executionTask)
		if err != nil {
			return "", fmt.Errorf("action execution failed: %w", err)
		}

		observation := extractString(actionResult)
		log.Printf("✅ Result obtained: %s", observation)
		observations = append(observations, observation)
	}

	// Step 3: Synthesis Phase
	synthesisPrompt := fmt.Sprintf(`Combine your historical insights and step-by-step tool observations to build a final human-readable answer. The response should be strictly human readable paragraph and concise.
Observations: %v
Original User Query: %s`, observations, input)

	log.Printf("⚙️ Synthesizing observation: %s", observations)
	finalResult, err := llms.GenerateFromSinglePrompt(ctx, w.textAgent, synthesisPrompt)
	if err != nil {
		return "", fmt.Errorf("final logic synthesis failed: %w", err)
	}

	return extractString(finalResult), nil
}

// SaveInteractionToVectorDB maps the exchange AND injects the parent link token to construct the implicit graph
func SaveInteractionToVectorDB(ctx context.Context, embeddingModel string, qdrantClient *qdrant.Client, sessionID string, prompt string, response string, parentVectorID string) (string, error) {
	combinedText := fmt.Sprintf("User: %s\nAssistant: %s", prompt, response)

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return "", fmt.Errorf("failed to init ollama client: %w", err)
	}

	req := &api.EmbedRequest{
		Model: embeddingModel,
		Input: combinedText,
	}

	resp, err := client.Embed(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ollama embedding generation failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return "", fmt.Errorf("ollama returned an empty embedding matrix")
	}
	vector := resp.Embeddings[0]

	pointID := uuid.New().String()

	payloadMap := map[string]interface{}{
		"session_id":            sessionID,
		"prompt":                prompt,
		"response":              response,
		"created_at":            time.Now().Unix(),
		"parent_interaction_id": parentVectorID,
	}

	points := []*qdrant.PointStruct{
		{
			Id:      qdrant.NewIDUUID(pointID),
			Vectors: qdrant.NewVectors(vector...),
			Payload: qdrant.NewValueMap(payloadMap),
		},
	}

	_, err = qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: "chat_history",
		Points:         points,
	})
	if err != nil {
		return "", fmt.Errorf("qdrant point upsert failed: %w", err)
	}

	return pointID, nil
}

// EmbedQuery turns a search string into a float32 vector slice
func EmbedQuery(ctx context.Context, modelName string, text string) ([]float32, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to init ollama client: %w", err)
	}

	req := &api.EmbedRequest{
		Model: modelName,
		Input: text,
	}

	resp, err := client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("embedding query generation failed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding matrix")
	}

	return resp.Embeddings[0], nil
}

// Internal helper for clean reflective parsing extraction of raw responses
func extractString(response interface{}) string {
	if response == nil {
		return ""
	}
	if str, ok := response.(string); ok {
		return str
	}

	v := reflect.ValueOf(response)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		f := v.FieldByName("Text")
		if f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}

	return fmt.Sprintf("%+v", response)
}
