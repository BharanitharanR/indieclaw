package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type LLMResultCoalator struct {
	llm llms.Model
}

func NewLLMResultCoalator(llm llms.Model) *LLMResultCoalator {
	return &LLMResultCoalator{
		llm: llm,
	}
}

// Coalesce synthesizes step results using persona-aware formatting
func (rc *LLMResultCoalator) Coalesce(ctx context.Context, steps []StepResult, persona *PersonaConfig) (string, error) {
	if len(steps) == 0 {
		return "", fmt.Errorf("no step results to coalesce")
	}

	log.Printf("🔗 Coalescing %d step results into final response", len(steps))

	prompt := rc.buildSynthesisPrompt(steps, persona)

	response, err := llms.GenerateFromSinglePrompt(ctx, rc.llm, prompt)
	if err != nil {
		log.Printf("❌ Synthesis LLM call failed: %v", err)
		return "", err
	}

	log.Printf("✅ Final response synthesized (%d chars)", len(response))

	finalResponse := rc.applyPersonaStyle(response, persona)

	return finalResponse, nil
}

// CoalesceWithDefinition uses the new PersonaDefinition type
func (rc *LLMResultCoalator) CoalesceWithDefinition(ctx context.Context, steps []StepResult, personaDef *PersonaDefinition, mode string) (string, error) {
	if len(steps) == 0 {
		return "", fmt.Errorf("no step results to coalesce")
	}

	log.Printf("[7/7] 🔗 RESULT COALATION")
	log.Printf("    Persona: %s | Mode: %s | Steps: %d", personaDef.Name, mode, len(steps))

	// Build synthesis prompt
	prompt := rc.buildSynthesisPromptNew(steps, personaDef, mode)

	response, err := llms.GenerateFromSinglePrompt(ctx, rc.llm, prompt)
	if err != nil {
		log.Printf("    ❌ Synthesis failed: %v", err)
		return "", err
	}

	log.Printf("    ✅ Response generated (%d chars)", len(response))

	// Apply persona-specific formatting
	finalResponse := rc.applyPersonaFormatting(response, personaDef, mode)

	return finalResponse, nil
}

// buildSynthesisPromptNew creates persona-aware synthesis prompt
func (rc *LLMResultCoalator) buildSynthesisPromptNew(steps []StepResult, personaDef *PersonaDefinition, mode string) string {
	var builder strings.Builder

	// Get response rule for this mode
	responseRule := personaDef.ResponseRules[mode]

	builder.WriteString(fmt.Sprintf("You are a %s assistant (v%d).\n\n", personaDef.Name, personaDef.Version))

	builder.WriteString("Your communication style:\n")
	builder.WriteString(fmt.Sprintf("- Tone: %s\n", responseRule.Tone))
	builder.WriteString(fmt.Sprintf("- Response ratios: Questions=%d%% | Reflection=%d%% | Advice=%d%% | Silence=%d%%\n",
		int(responseRule.QuestionsRatio*100),
		int(responseRule.ReflectionRatio*100),
		int(responseRule.AdviceRatio*100),
		int(responseRule.SilenceRatio*100),
	))

	builder.WriteString("\nStep Results to Synthesize:\n")
	builder.WriteString("==========================\n")

	for i, step := range steps {
		builder.WriteString(fmt.Sprintf("\nStep %d:\n", i+1))

		if step.Success {
			builder.WriteString(fmt.Sprintf("✓ %s\n", step.Result))
			if len(step.InternetData) > 0 {
				builder.WriteString(fmt.Sprintf("  (with %d sources)\n", len(step.InternetData)))
			}
		} else {
			builder.WriteString(fmt.Sprintf("✗ Failed: %s\n", step.Error))
		}
	}

	builder.WriteString("\n==========================\n\n")

	builder.WriteString("Requirements:\n")
	builder.WriteString(fmt.Sprintf("1. Maximum length: %d characters\n", responseRule.MaxLength))
	builder.WriteString(fmt.Sprintf("2. Keep tone: %s\n", responseRule.Tone))

	if responseRule.QuestionsRatio > 0.3 {
		builder.WriteString("3. Include thoughtful questions to guide thinking\n")
	}
	if responseRule.ReflectionRatio > 0.2 {
		builder.WriteString("4. Reflect back what you're hearing\n")
	}
	if len(responseRule.ForbiddenPatterns) > 0 {
		builder.WriteString(fmt.Sprintf("5. AVOID these patterns: %s\n", strings.Join(responseRule.ForbiddenPatterns, ", ")))
	}

	builder.WriteString("\nCreate a natural, flowing response that synthesizes all information above.\n")

	return builder.String()
}

// applyPersonaFormatting applies persona-specific response formatting
func (rc *LLMResultCoalator) applyPersonaFormatting(response string, personaDef *PersonaDefinition, mode string) string {
	result := response

	responseRule, ok := personaDef.ResponseRules[mode]
	if !ok {
		log.Printf("    ⚠️  No response rule for mode %q, using defaults", mode)
		return result
	}

	// Trim to max length
	if responseRule.MaxLength > 0 && len(result) > responseRule.MaxLength {
		log.Printf("    📏 Trimming from %d to %d chars", len(result), responseRule.MaxLength)
		result = result[:responseRule.MaxLength]

		// Try to cut at word boundary
		lastSpace := strings.LastIndex(result, " ")
		if lastSpace > 0 && lastSpace > responseRule.MaxLength-50 {
			result = result[:lastSpace]
		}
		if !strings.HasSuffix(result, "?") && !strings.HasSuffix(result, ".") {
			result = result + "..."
		}
	}

	// Validate against forbidden patterns
	for _, pattern := range responseRule.ForbiddenPatterns {
		if strings.Contains(strings.ToLower(result), strings.ToLower(pattern)) {
			log.Printf("    ⚠️  Response contains forbidden pattern: %q", pattern)
		}
	}

	// Ensure required elements are present
	for _, required := range responseRule.RequiredElements {
		if !strings.Contains(strings.ToLower(result), strings.ToLower(required)) {
			log.Printf("    ⚠️  Response missing required element: %q", required)
		}
	}

	return result
}

func (rc *LLMResultCoalator) buildSynthesisPrompt(steps []StepResult, persona *PersonaConfig) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("You are a %s assistant. Synthesize these step results into a coherent, unified response.\n\n", persona.Name))

	builder.WriteString("Step Results:\n")
	builder.WriteString("=============\n")

	for _, step := range steps {
		builder.WriteString(fmt.Sprintf("\nStep %d Result:\n", step.StepID))

		if step.Success {
			builder.WriteString(fmt.Sprintf("Status: SUCCESS\n"))
			builder.WriteString(fmt.Sprintf("Result: %s\n", step.Result))

			if len(step.InternetData) > 0 {
				builder.WriteString(fmt.Sprintf("Internet Data Included: %d sources\n", len(step.InternetData)))
			}
		} else {
			builder.WriteString(fmt.Sprintf("Status: FAILED\n"))
			builder.WriteString(fmt.Sprintf("Error: %s\n", step.Error))
		}
	}

	builder.WriteString("\n=============\n\n")

	builder.WriteString(fmt.Sprintf("Persona Instructions:\n%s\n\n", persona.PromptTemplate))

	builder.WriteString("Requirements:\n")
	builder.WriteString(fmt.Sprintf("1. Match the %s persona's tone (%s)\n", persona.Name, persona.Tone))
	builder.WriteString(fmt.Sprintf("2. Response style: %s\n", persona.ResponseStyle))
	builder.WriteString(fmt.Sprintf("3. Max length: %d characters\n", persona.MaxResponseLength))
	if persona.IncludeFollowup {
		builder.WriteString("4. Include a follow-up question at the end\n")
	}
	builder.WriteString("\nSynthesize all results into a single, coherent response that flows naturally.\n")
	builder.WriteString("Prioritize the most relevant and recent information from internet sources.\n")

	return builder.String()
}

func (rc *LLMResultCoalator) applyPersonaStyle(response string, persona *PersonaConfig) string {
	result := response

	if persona.MaxResponseLength > 0 && len(result) > persona.MaxResponseLength {
		log.Printf("📏 Trimming response to %d characters", persona.MaxResponseLength)
		result = result[:persona.MaxResponseLength]

		lastSpace := strings.LastIndex(result, " ")
		if lastSpace > 0 && lastSpace > persona.MaxResponseLength-100 {
			result = result[:lastSpace] + "..."
		}
	}

	if persona.IncludeFollowup && !strings.Contains(result, "?") {
		followUp := "\n\nWhat specific aspect would you like me to dive deeper into?"
		result = result + followUp
	}

	return result
}

type ResponseValidator struct {
	minLength int
	maxLength int
}

func NewResponseValidator() *ResponseValidator {
	return &ResponseValidator{
		minLength: 50,
		maxLength: 5000,
	}
}

func (rv *ResponseValidator) Validate(response string, persona *PersonaConfig) (bool, []string) {
	var errors []string

	if len(response) < rv.minLength {
		errors = append(errors, fmt.Sprintf("Response too short: %d chars (minimum: %d)", len(response), rv.minLength))
	}

	if len(response) > rv.maxLength {
		errors = append(errors, fmt.Sprintf("Response too long: %d chars (maximum: %d)", len(response), rv.maxLength))
	}

	if persona.MaxResponseLength > 0 && len(response) > persona.MaxResponseLength {
		errors = append(errors, fmt.Sprintf("Exceeds persona max length: %d > %d", len(response), persona.MaxResponseLength))
	}

	return len(errors) == 0, errors
}

type ResponseFormatter struct {
	medium string
}

func NewResponseFormatter(medium string) *ResponseFormatter {
	return &ResponseFormatter{
		medium: medium,
	}
}

func (rf *ResponseFormatter) Format(response string) string {
	switch rf.medium {
	case "whatsapp":
		return rf.formatForWhatsApp(response)
	case "json":
		return rf.formatAsJSON(response)
	case "text":
		return rf.formatAsText(response)
	default:
		return response
	}
}

func (rf *ResponseFormatter) formatForWhatsApp(response string) string {
	const waLimit = 4096

	if len(response) <= waLimit {
		return response
	}

	log.Printf("⚠️  Response exceeds WhatsApp limit (%d > %d), truncating", len(response), waLimit)

	truncated := response[:waLimit-10] + "..."

	lastPeriod := strings.LastIndex(truncated, ".")
	if lastPeriod > waLimit-500 {
		truncated = truncated[:lastPeriod+1]
	}

	return truncated
}

func (rf *ResponseFormatter) formatAsJSON(response string) string {
	data := map[string]interface{}{
		"response": response,
		"medium":   "json",
	}

	jsonBytes, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonBytes)
}

func (rf *ResponseFormatter) formatAsText(response string) string {
	return strings.TrimSpace(response)
}
