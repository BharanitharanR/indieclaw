package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type LLMPlanner struct {
	llm llms.Model
}

func NewLLMPlanner(llm llms.Model) *LLMPlanner {
	return &LLMPlanner{
		llm: llm,
	}
}

func (p *LLMPlanner) Plan(ctx context.Context, input string, context string, persona *PersonaConfig) (PlannerResponse, error) {
	if persona == nil {
		return PlannerResponse{}, fmt.Errorf("persona is required")
	}

	log.Printf("🤔 Planner: Analyzing request feasibility")

	// Use persona's planner prompt if available, otherwise use default
	plannerPromptTemplate := persona.PlannerPrompt
	if plannerPromptTemplate == "" {
		plannerPromptTemplate = defaultPlannerPrompt()
	}

	// Format with persona context
	planningPrompt := fmt.Sprintf(plannerPromptTemplate, input, persona.PromptTemplate)

	resp, err := llms.GenerateFromSinglePrompt(ctx, p.llm, planningPrompt)
	if err != nil {
		log.Printf("❌ Planner LLM call failed: %v", err)
		return PlannerResponse{}, err
	}

	log.Printf("📋 Planner response received (%d chars)", len(resp))

	planResp := p.parseResponse(resp)
	log.Printf("✅ Parsed: can_help=%v, confidence=%.0f%%", planResp.CanHelp, planResp.Confidence)

	return planResp, nil
}

func (p *LLMPlanner) parseResponse(response string) PlannerResponse {
	result := PlannerResponse{
		CanHelp:    false,
		Confidence: 0,
		Reasoning:  "Unable to parse planner response",
	}

	jsonStr := p.extractJSON(response)
	if jsonStr == "" {
		log.Printf("⚠️  No JSON found in planner response")
		return result
	}

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &data)
	if err != nil {
		log.Printf("⚠️  Failed to parse JSON: %v", err)
		return result
	}

	if v, ok := data["can_help"].(bool); ok {
		result.CanHelp = v
	}

	if v, ok := data["confidence"].(float64); ok {
		result.Confidence = v
	}

	if v, ok := data["reasoning"].(string); ok {
		result.Reasoning = v
	}

	if v, ok := data["pii_warning"].(string); ok && v != "" {
		result.PiiWarning = v
	}

	if v, ok := data["steps"].([]interface{}); ok {
		for _, step := range v {
			if s, ok := step.(string); ok {
				result.Steps = append(result.Steps, s)
			}
		}
	}

	return result
}

func (p *LLMPlanner) extractJSON(text string) string {
	re := regexp.MustCompile(`\{[^{}]*(?:\{[^{}]*\}[^{}]*)*\}`)
	matches := re.FindStringIndex(text)

	if matches == nil {
		return ""
	}

	return text[matches[0]:matches[1]]
}

type ConfidenceValidator struct {
	threshold float64
	piiAlert  bool
}

func NewConfidenceValidator(threshold float64) *ConfidenceValidator {
	return &ConfidenceValidator{
		threshold: threshold,
		piiAlert:  true,
	}
}

func (cv *ConfidenceValidator) ShouldProceed(confidence float64, hasPII bool) bool {
	if hasPII && cv.piiAlert {
		log.Printf("⚠️  PII detected in request - requiring higher confidence")
		return confidence >= (cv.threshold + 20)
	}

	return confidence >= cv.threshold
}

func (cv *ConfidenceValidator) RejectionMessage() string {
	return "I may not be able to help with this request. Could you provide more details or rephrase your question?"
}

type StepRefinement struct {
	llm llms.Model
}

func NewStepRefinement(llm llms.Model) *StepRefinement {
	return &StepRefinement{
		llm: llm,
	}
}

func (sr *StepRefinement) RefineSteps(ctx context.Context, roughSteps []string, input string) ([]ExecutionStep, error) {
	log.Printf("🔧 Refining %d planner steps into execution steps", len(roughSteps))

	var executionSteps []ExecutionStep

	for i, step := range roughSteps {
		prompt := fmt.Sprintf(`For this task step: "%s"

Given the user request: "%s"

Determine:
1. What tools are needed (web_search, calculator, database, etc.)?
2. What is the expected output?
3. Any internet search needed for latest data?

Respond with one-word tool names separated by commas.
`, step, input)

		resp, err := llms.GenerateFromSinglePrompt(ctx, sr.llm, prompt)
		if err != nil {
			log.Printf("⚠️  Failed to refine step %d: %v", i+1, err)
			executionSteps = append(executionSteps, ExecutionStep{
				ID:          i + 1,
				Description: step,
				Action:      step,
				Tools:       []string{"web_search"},
			})
			continue
		}

		tools := parseTools(resp)
		if len(tools) == 0 {
			tools = []string{"web_search"}
		}

		executionSteps = append(executionSteps, ExecutionStep{
			ID:          i + 1,
			Description: step,
			Action:      step,
			Tools:       tools,
		})

		log.Printf("   Step %d: %s (tools: %v)", i+1, step, tools)
	}

	log.Printf("✅ Refined into %d execution steps", len(executionSteps))
	return executionSteps, nil
}

func parseTools(response string) []string {
	parts := strings.FieldsFunc(response, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})

	var tools []string
	for _, p := range parts {
		tool := strings.TrimSpace(strings.ToLower(p))
		if tool != "" && tool != "and" && tool != "or" {
			tools = append(tools, tool)
		}
	}

	return tools
}
