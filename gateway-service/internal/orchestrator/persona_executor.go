package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

// PersonaExecutorImpl implements the PersonaExecutor interface
type PersonaExecutorImpl struct {
	llm llms.Model
}

// NewPersonaExecutor creates a new persona executor
func NewPersonaExecutor(llm llms.Model) PersonaExecutor {
	return &PersonaExecutorImpl{llm: llm}
}

// ClassifyIntent classifies a user message into one of the persona's intents
func (pe *PersonaExecutorImpl) ClassifyIntent(message string, pctx *PersonaContext) (string, float32, error) {
	if pctx.Definition == nil || len(pctx.Definition.Intents) == 0 {
		return "unknown", 0.0, fmt.Errorf("no persona definition or intents available")
	}

	// Build intent options from persona
	intentNames := make([]string, 0, len(pctx.Definition.Intents))
	for name := range pctx.Definition.Intents {
		intentNames = append(intentNames, name)
	}

	// Create classification prompt
	prompt := fmt.Sprintf(`You are an expert at classifying user intents.

User message: "%s"

Available intent categories for this assistant:
%s

Classify the user's message into ONE of these categories. Consider the semantic meaning, not just keywords.

Respond with ONLY:
INTENT: [category_name]
CONFIDENCE: [0-100]

Be strict. If the message doesn't clearly fit an intent, use lower confidence. If it's outside the scope of these intents, use "unknown" with low confidence.`,
		message,
		formatIntentList(pctx.Definition.Intents),
	)

	response, err := pe.llm.GenerateContent(context.Background(), []llms.MessageContent{
		{Role: "user", Parts: []llms.ContentPart{llms.TextContent{Text: prompt}}},
	})
	if err != nil {
		return "unknown", 0.0, fmt.Errorf("intent classification LLM call failed: %w", err)
	}

	if len(response.Choices) == 0 {
		return "unknown", 0.0, fmt.Errorf("LLM returned no response")
	}

	// Parse response
	content := response.Choices[0].Content
	intent, confidence := parseIntentResponse(content, intentNames)

	log.Printf("    Intent: %s (%.0f%% confidence)", intent, confidence*100)

	return intent, confidence, nil
}

// GetModeForIntent returns the response mode for a given intent
func (pe *PersonaExecutorImpl) GetModeForIntent(intent string, persona *PersonaDefinition) PersonaRule {
	if rule, ok := persona.Intents[intent]; ok {
		return rule
	}

	// Return empty rule if not found
	return PersonaRule{
		Mode:  "unknown",
		Depth: "light",
	}
}

// ValidateResponseAgainstRules checks if a response violates any persona rules
func (pe *PersonaExecutorImpl) ValidateResponseAgainstRules(response string, mode string, persona *PersonaDefinition) []ValidationIssue {
	var issues []ValidationIssue

	// Get response rule for this mode
	rule, ok := persona.ResponseRules[mode]
	if !ok {
		// No specific rule for this mode, skip validation
		return issues
	}

	// Check length constraint
	if len(response) > rule.MaxLength {
		issues = append(issues, ValidationIssue{
			Gate:     "response_length",
			Severity: "warning",
			Message:  fmt.Sprintf("Response exceeds max length (%d > %d)", len(response), rule.MaxLength),
			Fix:      "Reduce response length",
		})
	}

	// Check for forbidden patterns
	for _, pattern := range rule.ForbiddenPatterns {
		if strings.Contains(strings.ToLower(response), strings.ToLower(pattern)) {
			issues = append(issues, ValidationIssue{
				Gate:     "forbidden_pattern",
				Severity: "error",
				Message:  fmt.Sprintf("Response contains forbidden pattern: %q", pattern),
				Fix:      fmt.Sprintf("Remove or rephrase the phrase %q", pattern),
			})
		}
	}

	// Check for required elements
	for _, required := range rule.RequiredElements {
		if !strings.Contains(strings.ToLower(response), strings.ToLower(required)) {
			issues = append(issues, ValidationIssue{
				Gate:     "missing_element",
				Severity: "warning",
				Message:  fmt.Sprintf("Response missing required element: %q", required),
				Fix:      fmt.Sprintf("Ensure response includes %q", required),
			})
		}
	}

	// Check validation gates
	for _, gate := range persona.ValidationGates {
		if issue := validateGate(response, gate); issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// ApplyResponseTemplate applies a response template with data substitution
func (pe *PersonaExecutorImpl) ApplyResponseTemplate(template string, data map[string]interface{}) string {
	result := template

	// Simple template substitution: {{key}} -> value
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}

// classifyIntentWithRules uses persona intent rules to classify
// This is a simpler, rule-based approach before LLM classification
func (pe *PersonaExecutorImpl) classifyIntentWithRules(message string, persona *PersonaDefinition) (string, float32) {
	message = strings.ToLower(message)
	bestMatch := "unknown"
	bestScore := float32(0.0)

	// Check if message matches any intent's description or keywords
	// (This would be enhanced with a proper keyword/pattern system)
	for intentName := range persona.Intents {
		// Quick heuristic: check if intent mode gives us a clue
		if strings.Contains(message, intentName) {
			score := float32(0.6)
			if score > bestScore {
				bestMatch = intentName
				bestScore = score
			}
		}
	}

	return bestMatch, bestScore
}

// Helper: format intent list for prompt
func formatIntentList(intents map[string]PersonaRule) string {
	var items []string
	for name, rule := range intents {
		items = append(items, fmt.Sprintf("- %s (mode: %s, depth: %s)", name, rule.Mode, rule.Depth))
	}
	return strings.Join(items, "\n")
}

// Helper: parse LLM response for intent and confidence
func parseIntentResponse(content string, validIntents []string) (string, float32) {
	lines := strings.Split(content, "\n")
	var intent string
	var confidence float32 = 0.0

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "INTENT:") {
			intent = strings.TrimSpace(strings.TrimPrefix(line, "INTENT:"))
		}
		if strings.HasPrefix(strings.TrimSpace(line), "CONFIDENCE:") {
			confStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			// Try to parse as percentage
			var conf int
			fmt.Sscanf(confStr, "%d", &conf)
			confidence = float32(conf) / 100.0
		}
	}

	// Validate intent is in the valid list
	found := false
	for _, valid := range validIntents {
		if strings.ToLower(intent) == strings.ToLower(valid) {
			intent = valid
			found = true
			break
		}
	}

	if !found {
		// Intent not recognized, use "unknown"
		intent = "unknown"
		confidence = 0.2
	}

	return intent, confidence
}

// Helper: validate a single gate
func validateGate(response string, gate ValidationGate) *ValidationIssue {
	switch gate.GateType {
	case "has_element":
		element, ok := gate.Parameters["element"].(string)
		if !ok {
			return nil
		}
		if !strings.Contains(strings.ToLower(response), strings.ToLower(element)) {
			return &ValidationIssue{
				Gate:     gate.Name,
				Severity: "error",
				Message:  fmt.Sprintf("Missing required element: %s", element),
				Fix:      gate.ErrorMessage,
			}
		}

	case "length_check":
		maxLen, ok := gate.Parameters["max_length"].(float64)
		if !ok {
			return nil
		}
		if len(response) > int(maxLen) {
			return &ValidationIssue{
				Gate:     gate.Name,
				Severity: "error",
				Message:  fmt.Sprintf("Response too long: %d > %d", len(response), int(maxLen)),
				Fix:      gate.ErrorMessage,
			}
		}

	case "pattern_check":
		pattern, ok := gate.Parameters["forbidden_pattern"].(string)
		if !ok {
			return nil
		}
		if strings.Contains(strings.ToLower(response), strings.ToLower(pattern)) {
			return &ValidationIssue{
				Gate:     gate.Name,
				Severity: "error",
				Message:  fmt.Sprintf("Contains forbidden pattern: %s", pattern),
				Fix:      gate.ErrorMessage,
			}
		}
	}

	return nil
}
