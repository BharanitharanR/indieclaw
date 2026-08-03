package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

type SimpleTokenManager struct {
	avgCharsPerToken int
}

func NewSimpleTokenManager() *SimpleTokenManager {
	return &SimpleTokenManager{
		avgCharsPerToken: 4,
	}
}

func (tm *SimpleTokenManager) CountTokens(text string) (int, error) {
	if text == "" {
		return 0, nil
	}
	tokens := (len(text) + tm.avgCharsPerToken - 1) / tm.avgCharsPerToken
	return tokens, nil
}

func (tm *SimpleTokenManager) TrimContext(context string, maxTokens int) (string, error) {
	if context == "" {
		return "", nil
	}

	tokens, _ := tm.CountTokens(context)
	log.Printf("📊 Context tokens: %d | Max allowed: %d", tokens, maxTokens)

	if tokens <= maxTokens {
		return context, nil
	}

	maxChars := maxTokens * tm.avgCharsPerToken
	log.Printf("   Trimming to %d characters", maxChars)

	if len(context) > maxChars {
		return context[:maxChars], nil
	}

	return context, nil
}

func (tm *SimpleTokenManager) EstimateTotalTokens(input string, context string, persona *PersonaConfig) (int, error) {
	inputTokens, _ := tm.CountTokens(input)
	contextTokens, _ := tm.CountTokens(context)

	personaPromptTokens, _ := tm.CountTokens(persona.PromptTemplate)

	total := inputTokens + contextTokens + personaPromptTokens

	log.Printf("📈 Token Estimation:")
	log.Printf("   Input: %d tokens", inputTokens)
	log.Printf("   Context: %d tokens", contextTokens)
	log.Printf("   Persona: %d tokens", personaPromptTokens)
	log.Printf("   Total: %d tokens", total)

	return total, nil
}

type ContextWindow struct {
	MaxContextTokens int
	ReservedForOutput int
	ReservedForPlan  int
	AvailableTokens  int
}

func NewContextWindow(maxModelTokens int) *ContextWindow {
	reserved := 1000
	planReserved := 500

	return &ContextWindow{
		MaxContextTokens: maxModelTokens - reserved - planReserved,
		ReservedForOutput: reserved,
		ReservedForPlan: planReserved,
		AvailableTokens: maxModelTokens - reserved - planReserved,
	}
}

func (cw *ContextWindow) IsWithinBudget(totalTokens int) bool {
	return totalTokens <= (cw.MaxContextTokens + cw.ReservedForOutput + cw.ReservedForPlan)
}

type ContextCompressor struct {
	tm *SimpleTokenManager
}

func NewContextCompressor() *ContextCompressor {
	return &ContextCompressor{
		tm: NewSimpleTokenManager(),
	}
}

func (cc *ContextCompressor) CompressContexts(contexts []RetrievedContext, maxTokens int, minSimilarity float64) (string, error) {
	if len(contexts) == 0 {
		return "", nil
	}

	var selected []RetrievedContext
	var totalTokens int

	log.Printf("📦 Compressing %d contexts into %d tokens (min similarity: %.2f)", len(contexts), maxTokens, minSimilarity)

	for _, ctx := range contexts {
		if ctx.Similarity < minSimilarity {
			continue
		}

		ctxTokens, _ := cc.tm.CountTokens(ctx.Content)
		if totalTokens+ctxTokens > maxTokens {
			log.Printf("   Stopping: adding context would exceed token budget (%d + %d > %d)", totalTokens, ctxTokens, maxTokens)
			break
		}

		selected = append(selected, ctx)
		totalTokens += ctxTokens
		log.Printf("   [%.2f] Added context: %d tokens", ctx.Similarity, ctxTokens)
	}

	var builder strings.Builder
	builder.WriteString("=== RETRIEVED CONTEXT ===\n")

	for i, ctx := range selected {
		builder.WriteString(fmt.Sprintf("\n[Context %d] (similarity: %.2f)\n", i+1, ctx.Similarity))
		builder.WriteString(ctx.Content)
		builder.WriteString("\n")
	}

	builder.WriteString("\n=== END CONTEXT ===\n")

	result := builder.String()
	log.Printf("✅ Compressed %d contexts into %d tokens", len(selected), totalTokens)

	return result, nil
}

func SerializeContext(contexts []RetrievedContext) string {
	data := map[string]interface{}{
		"context_count": len(contexts),
		"contexts": contexts,
	}

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("❌ Failed to serialize context: %v", err)
		return ""
	}

	return string(jsonBytes)
}
