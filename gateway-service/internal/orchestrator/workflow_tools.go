package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// ToolExecutionStep represents a single tool invocation in the ReAct loop
type ToolExecutionStep struct {
	StepID   int                    `json:"step_id"`
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

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

// InitializeToolRegistry sets up all available tools for the workflow
func (w *AgentWorkflow) InitializeToolRegistry() *ToolRegistry {
	registry := NewToolRegistry()

	// Register built-in tools
	_ = registry.Register(WebSearchTool())
	_ = registry.Register(MathTool())
	_ = registry.Register(FileFetchTool())
	_ = registry.Register(HTTPCallTool())
	_ = registry.Register(JSONParseTool())

	// TODO: Register custom domain-specific tools here
	// Example:
	// _ = registry.Register(YourCustomDatabaseQueryTool())
	// _ = registry.Register(YourCustomDataTransformTool())

	log.Printf("✅ Tool registry initialized with %d tools", len(registry.tools))
	return registry
}

// RunWithTools executes the core ReAct loop WITH tool support
func (w *AgentWorkflow) RunWithTools(ctx context.Context, sessionID string, input string, isVision bool, imageBase64 string, toolRegistry *ToolRegistry) (string, error) {
	if isVision {
		// Vision path remains unchanged
		return w.Run(ctx, sessionID, input, isVision, imageBase64)
	}

	// --- TEXT AGENT: REACT LOOP WITH TOOLS ---

	// Step 1: Enhanced planning that can identify tools
	planPrompt := fmt.Sprintf(`You are an intelligent Assistant Planner with access to external tools.

AVAILABLE TOOLS:
%s

Your goal is to answer the user or perform the task using tools when necessary.

RULES:
- If the task requires current web info, use "web_search"
- If the task requires calculations, use "calculate"
- If the task requires file access, use "fetch_file"
- If the task requires API calls, use "http_call"
- If the task requires JSON parsing, use "parse_json"
- For general knowledge or analysis, proceed directly (no tool needed)
- Only invoke tools when genuinely necessary

Output your execution path STRICTLY as JSON:
{
  "steps": [
    {
      "id": 1,
      "action": "Description of what to do",
      "tool_name": "web_search",
      "tool_input": {"query": "search term"},
      "needs_user_input": false
    }
  ]
}

User Prompt: "%s"`, w.formatToolsForPrompt(toolRegistry), input)

	rawPlanResp, err := w.textAgent.Generate(ctx, sessionID, planPrompt)
	if err != nil {
		return "", fmt.Errorf("planning phase generation failed: %w", err)
	}

	planString := extractString(rawPlanResp)
	log.Printf("📋 Generated plan: %s", planString)

	var plan ReActPlanWithTools
	if err := json.Unmarshal([]byte(planString), &plan); err != nil {
		log.Printf("WARN: Failed parsing plan JSON. Fallback to direct resolution. Error: %v Raw: %s", err, planString)
		directResp, err := w.textAgent.Generate(ctx, sessionID, input)
		return extractString(directResp), err
	}

	var observations []string
	var toolExecutions []ToolExecutionStep

	// Step 2: Execute steps with tool support
	for _, step := range plan.Steps {
		log.Printf("⚙️ ReAct Step [%d]: %s (Tool: %s, Needs Input: %t)", step.ID, step.Action, step.ToolName, step.NeedsUserInput)

		if step.NeedsUserInput {
			return fmt.Sprintf("⏸️ Paused for clarification: %s", step.Action), nil
		}

		var observation string
		var toolExec ToolExecutionStep

		// If a tool is specified, execute it
		if step.ToolName != "" {
			toolExec.StepID = step.ID
			toolExec.ToolName = step.ToolName
			toolExec.Input = step.ToolInput

			toolOutput, err := toolRegistry.Execute(ctx, step.ToolName, ToolInput{Args: step.ToolInput})
			toolExec.Result = toolOutput.Data
			if err != nil {
				toolExec.Error = err.Error()
				log.Printf("❌ Tool execution failed: %v", err)
			}

			toolExecutions = append(toolExecutions, toolExec)

			// Format tool result for LLM consumption
			observation = fmt.Sprintf("Tool '%s' returned: %+v", step.ToolName, toolOutput.Data)
		} else {
			// No tool: proceed with direct LLM generation
			executionTask := fmt.Sprintf("Task: %s. Provide the result or answer for this task.", step.Action)
			actionResult, err := w.textAgent.Generate(ctx, sessionID, executionTask)
			if err != nil {
				return "", fmt.Errorf("action execution failed: %w", err)
			}
			observation = extractString(actionResult)
		}

		log.Printf("✅ Observation: %s", observation)
		observations = append(observations, observation)
	}

	// Step 3: Synthesis Phase - combine all observations + tool results
	synthesisPrompt := fmt.Sprintf(`You are synthesizing a complete response based on tool results and reasoning.

TOOL EXECUTIONS:
%s

OBSERVATIONS:
%s

Original User Query: %s

Provide a clear, concise, human-readable response that:
1. Incorporates all tool results meaningfully
2. Answers the user's original query completely
3. Is formatted as a natural paragraph (not bullet points unless necessary)`,
		formatToolExecutions(toolExecutions),
		strings.Join(observations, "\n"),
		input)

	log.Printf("⚙️ Synthesizing final response...")
	finalResult, err := w.textAgent.Generate(ctx, sessionID, synthesisPrompt)
	if err != nil {
		return "", fmt.Errorf("final synthesis failed: %w", err)
	}

	return extractString(finalResult), nil
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

// ParseToolCallFromLLMOutput extracts tool calls from unstructured LLM responses
// Handles cases where the LLM returns tool calls in natural language or special syntax
func ParseToolCallFromLLMOutput(output string) (toolName string, input map[string]interface{}, found bool) {
	// Pattern 1: <tool>tool_name</tool><input>{"key": "value"}</input>
	toolPattern := regexp.MustCompile(`<tool>(\w+)</tool>`)
	inputPattern := regexp.MustCompile(`<input>(.*?)</input>`)

	toolMatch := toolPattern.FindStringSubmatch(output)
	inputMatch := inputPattern.FindStringSubmatch(output)

	if len(toolMatch) < 2 || len(inputMatch) < 2 {
		return "", nil, false
	}

	toolName = toolMatch[1]
	var parsedInput map[string]interface{}

	if err := json.Unmarshal([]byte(inputMatch[1]), &parsedInput); err != nil {
		log.Printf("WARN: Failed to parse tool input JSON: %v", err)
		return "", nil, false
	}

	return toolName, parsedInput, true
}

// ChainToolCalls allows sequential tool execution based on previous results
// Useful for workflows requiring multi-step tool coordination
func (w *AgentWorkflow) ChainToolCalls(
	ctx context.Context,
	sessionID string,
	toolRegistry *ToolRegistry,
	toolChain []map[string]interface{},
) ([]interface{}, error) {
	var results []interface{}

	for i, toolCall := range toolChain {
		toolName := toolCall["tool_name"].(string)
		toolArgs := toolCall["args"].(map[string]interface{})

		log.Printf("🔗 Chain Step %d: Executing %s", i+1, toolName)

		output, err := toolRegistry.Execute(ctx, toolName, ToolInput{Args: toolArgs})
		if err != nil {
			return nil, fmt.Errorf("chain step %d failed: %w", i+1, err)
		}

		results = append(results, output.Data)
	}

	return results, nil
}
