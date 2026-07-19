package orchestrator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gateway-service/internal/pkg/vision"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"

	"github.com/Protocol-Lattice/go-agent"
	"github.com/Protocol-Lattice/go-agent/src/memory"
	"github.com/Protocol-Lattice/go-agent/src/models"
	"github.com/ollama/ollama/api"
	"github.com/qdrant/go-client/qdrant"
)

type AgentWorkflow struct {
	textAgent   *agent.Agent
	visionAgent *agent.Agent
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

func NewAgentWorkflow(ctx context.Context, textModelName, visionModelName string) (*AgentWorkflow, error) {
	memBank := memory.NewMemoryBankWithStore(memory.NewInMemoryStore())
	mem := memory.NewSessionMemory(memBank, 8)

	// 1. Text Model initialization
	textModel, err := models.NewOllamaLLM(textModelName, "")
	if err != nil {
		return nil, err
	}

	// 2. Vision Model initialization (Manual creation + Wrapper)
	originalLLM, err := models.NewOllamaLLM(visionModelName, "")
	if err != nil {
		return nil, err
	}
	// Wrap it with your fixed logic
	fixedLLM := &vision.VisionFixedLLM{OllamaLLM: originalLLM}

	// 3. Text Agent
	t, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert orchestrator assistant capable of analyzing problems and formulating step-by-step processing paths.",
		Model:        textModel,
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	// 4. Vision Agent (Using the fixedLLM wrapper)
	v, err := agent.New(agent.Options{
		SystemPrompt: "You are an expert vision assistant.",
		Model:        fixedLLM,
		Memory:       mem,
	})
	if err != nil {
		return nil, err
	}

	return &AgentWorkflow{textAgent: t, visionAgent: v}, nil
}

// Run executes the core ReAct planning loop. If it encounters a clarification pause, it yields execution gracefully.
func (w *AgentWorkflow) Run(ctx context.Context, sessionID string, input string, isVision bool, imageBase64 string) (string, error) {
	if isVision {
		// 1. Decode the Base64 string into RAW BYTES
		imgBytes, err := base64.StdEncoding.DecodeString(imageBase64)
		if err != nil {
			return "", fmt.Errorf("failed to decode image: %v", err)
		}
		mimeType := http.DetectContentType(imgBytes)
		log.Printf("DEBUG: Detected MIME for file: %s", mimeType)

		// 2. Prepare the file structure with RAW BYTES
		files := []models.File{
			{
				Name: "input.jpg",
				MIME: "image/jpeg",
				Data: imgBytes,
			},
		}

		// 3. Invoke the library method directly for Vision tasks
		return w.visionAgent.GenerateWithFiles(ctx, sessionID, input, files)
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

	rawPlanResp, err := w.textAgent.Generate(ctx, sessionID, planPrompt)
	if err != nil {
		return "", fmt.Errorf("planning phase generation failed: %w", err)
	}

	// Safely clean and extract string if returned as raw response wrapper
	planString := extractString(rawPlanResp)

	var plan ReActPlan
	if err := json.Unmarshal([]byte(planString), &plan); err != nil {
		log.Printf("WARN: Failed parsing model plan JSON. Fallback to direct resolution. Error: %v Raw: %s", err, planString)
		// Fallback graceful degradation: process directly if JSON boundaries break
		directResp, err := w.textAgent.Generate(ctx, sessionID, input)
		return extractString(directResp), err
	}

	var observations []string

	// Step 2: Iterate through steps sequentially (Stateless execution boundaries)
	for _, step := range plan.Steps {
		log.Printf("⚙️ ReAct Executor processing Step [%d]: %s (Needs Input: %t)", step.ID, step.Action, step.NeedsUserInput)

		if step.NeedsUserInput {
			// HUMAN-IN-THE-LOOP SUSPENSION: Halts loop execution immediately.
			// The current state is preserved via implicit graph linkages.
			return fmt.Sprintf("⏸️ Paused for clarification: %s", step.Action), nil
		}
		executionTask := fmt.Sprintf("Task: %s. Provide the result or answer for this task.", step.Action)

		// We call the agent again to actually perform the work
		actionResult, err := w.textAgent.Generate(ctx, sessionID, executionTask)
		if err != nil {
			return "", fmt.Errorf("action execution failed: %w", err)
		}

		// Now 'observation' actually contains the real work done by the AI
		observation := extractString(actionResult)

		log.Printf("✅ Result obtained: %s", observation)
		observations = append(observations, observation)

	}

	// Step 3: Synthesis Phase (Combine tracking context and observations into the final user deliverable)
	synthesisPrompt := fmt.Sprintf(`Combine your historical insights and step-by-step tool observations to build a final human-readable answer.The reposnse should be strictly humand readable paragraph and concise.
Observations: %v
Original User Query: %s`, observations, input)

	log.Printf("⚙️ Synthesizing observation: %s", observations)
	finalResult, err := w.textAgent.Generate(ctx, sessionID, synthesisPrompt)
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

	// Building metadata payload map including our structural implicit graph "Edge"
	payloadMap := map[string]interface{}{
		"session_id":            sessionID,
		"prompt":                prompt,
		"response":              response,
		"created_at":            time.Now().Unix(),
		"parent_interaction_id": parentVectorID, // <-- Graph edge binding link
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
