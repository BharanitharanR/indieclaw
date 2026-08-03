package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

// ReActPlanWithTools extends the original plan to support tool calls
type ReActPlanWithTools struct {
	Steps []ReActStepWithTools `json:"steps"`
}

type ReActStepWithTools struct {
	ID             int                    `json:"id"`
	Action         string                 `json:"action"`
	ToolName       string                 `json:"tool_name,omitempty"`  // Name of tool to call (if any)
	ToolInput      map[string]interface{} `json:"tool_input,omitempty"` // Tool parameters
	NeedsUserInput bool                   `json:"needs_user_input"`
}

// ToolExecutionStep represents a single tool invocation in the ReAct loop
type ToolExecutionStep struct {
	StepID   int                    `json:"step_id"`
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// InitializeToolRegistry sets up all available tools for the workflow
func (w *AgentWorkflow) InitializeToolRegistry() *ToolRegistry {
	registry := NewToolRegistry()
	log.Printf("✅ Tool registry initialized with %d tools", len(registry.tools))
	return registry
}

func (w *AgentWorkflow) RunWithTools(ctx context.Context, sessionID string, input string, isVision bool, imageBase64 string, toolRegistry *ToolRegistry) (string, error) {
	handler := NewAgentHandler()

	// 1. Handle Vision path
	if isVision {
		contentParts := []llms.ContentPart{
			llms.TextPart(input),
			llms.ImageURLPart("data:image/jpeg;base64," + imageBase64),
		}

		resp, err := w.visionAgent.GenerateContent(ctx, []llms.MessageContent{
			{Role: llms.ChatMessageTypeHuman, Parts: contentParts},
		})
		if err != nil {
			return "", err
		}
		if len(resp.Choices) > 0 {
			return resp.Choices[0].Content, nil
		}
		return "", fmt.Errorf("empty vision response")
	}

	// 2. Extract and Validate LangChain tools
	rawTools := toolRegistry.GetLangChainTools()
	var lcTools []tools.Tool
	for _, t := range rawTools {
		if toolItem, ok := t.(tools.Tool); ok {
			// CRITICAL: Log exact tool names so you can verify they match LLM action choices
			log.Printf("🛠️ Registered Tool -> Name: %q | Description: %q", toolItem.Name(), toolItem.Description())
			lcTools = append(lcTools, toolItem)
		}
	}

	if len(lcTools) == 0 {
		return "", fmt.Errorf("no valid langchain tools found in registry")
	}

	log.Printf("🚀 Initializing LangChain Agent Executor with %d tools...", len(lcTools))

	// Build a clear manifest of available tools for the prompt
	var toolDescriptions strings.Builder
	for _, t := range lcTools {
		toolDescriptions.WriteString(fmt.Sprintf("- Name: %s\n  Description: %s\n", t.Name(), t.Description()))
	}

	// Construct an explicit ReAct instruction set forcing the model to use the tools
	promptPrefix := fmt.Sprintf(`You are an advanced AI assistant that MUST use tools to answer questions.You are an offline model and is few years old than the current date.Never assume your infromation is correct without using the internet search. Never guess or rely solely on internal knowledge when tools are available.

		You have access to the following tools:
		%s

		To use a tool, you MUST use the following exact format:
		Thought: Do I need to use a tool? Yes.
		Action: the name of the tool to take, should be one of [%s]
		Action Input: the input to the tool
		Observation: the result of the action
		... (this Thought/Action/Action Input/Observation can repeat N times)
		Thought: I now know the final answer
		Final Answer: the final answer to the original input question

		Begin!`, toolDescriptions.String(), getToolNamesList(lcTools))

	// 3. Initialize the OneShotAgent with strict ReAct system prompt
	agent := agents.NewOneShotAgent(
		w.textAgent,
		lcTools,
		agents.WithPromptPrefix(promptPrefix),
		agents.WithMaxIterations(5),
		agents.WithCallbacksHandler(handler),
	)

	executor := agents.NewExecutor(agent)

	// 4. Execute the chain and dump all result keys for debugging
	result, err := chains.Call(ctx, executor, map[string]any{
		"input": input,
	})

	// Debug output keys returned by the executor
	if result != nil {
		var keys []string
		for k := range result {
			keys = append(keys, k)
		}
		log.Printf("📦 Executor Result Keys: %v", keys)
	}

	if err != nil {
		log.Printf("⚠️ LangChain execution failed (%v), attempting single-prompt fallback...", err)
		fallbackResp, fallbackErr := llms.GenerateFromSinglePrompt(ctx, w.textAgent, input)
		if fallbackErr != nil {
			return "", fmt.Errorf("langchain execution and fallback both failed: %w (fallback error: %v)", err, fallbackErr)
		}
		return fallbackResp, nil
	}

	// 5. Inspect tools used post-execution via intermediate steps
	if steps, ok := result["intermediate_steps"].([]schema.AgentStep); ok {
		log.Printf("🔍 Found %d intermediate tool execution steps", len(steps))
		for _, step := range steps {
			log.Printf("⚡ Tool Executed Successfully: [%s] with input [%s] -> Output: %s", step.Action.Tool, step.Action.ToolInput, step.Observation)
		}
	} else {
		log.Printf("⚠️ No 'intermediate_steps' found or type assertion failed in result map.")
	}

	// 6. Safely parse and return the final text result
	if outputStr, ok := result["output"].(string); ok {
		return outputStr, nil
	}
	if outputStr, ok := result["text"].(string); ok {
		return outputStr, nil
	}
	resultJSON, _ := json.Marshal(result)
	log.Printf("⚠️ Unexpected result format, returning raw JSON: %s", string(resultJSON))
	return string(resultJSON), nil
}

// formatToolsForPrompt creates a formatted list of available tools for the LLM
func (w *AgentWorkflow) formatToolsForPrompt(registry *ToolRegistry) string {
	var buf strings.Builder
	tools := registry.List()

	for _, tool := range tools {
		buf.WriteString(fmt.Sprintf("- %s: %s\n", tool["name"], tool["description"]))
	}

	return buf.String()
}

// formatToolExecutions creates a readable summary of tool executions
func formatToolExecutions(executions []ToolExecutionStep) string {
	if len(executions) == 0 {
		return "(No tools executed)"
	}

	var buf strings.Builder
	for _, exec := range executions {
		buf.WriteString(fmt.Sprintf("Step %d - %s:\n", exec.StepID, exec.ToolName))
		buf.WriteString(fmt.Sprintf("  Input: %+v\n", exec.Input))
		if exec.Error != "" {
			buf.WriteString(fmt.Sprintf("  Error: %s\n", exec.Error))
		} else {
			buf.WriteString(fmt.Sprintf("  Result: %+v\n", exec.Result))
		}
	}
	return buf.String()
}

func getToolNamesList(toolsList []tools.Tool) string {
	var names []string
	for _, t := range toolsList {
		names = append(names, t.Name())
	}
	return strings.Join(names, ", ")
}
