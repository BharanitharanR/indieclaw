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
