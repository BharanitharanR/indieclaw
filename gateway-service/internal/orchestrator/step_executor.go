package orchestrator

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
)

type LLMStepExecutor struct {
	llm           llms.Model
	searcher      InternetSearcher
	maxRetries    int
	retryDelay    time.Duration
}

func NewLLMStepExecutor(llm llms.Model, searcher InternetSearcher) *LLMStepExecutor {
	return &LLMStepExecutor{
		llm:        llm,
		searcher:   searcher,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

func (se *LLMStepExecutor) ExecuteStep(ctx context.Context, step ExecutionStep, tools map[string]Tool) (StepResult, error) {
	result := StepResult{
		StepID:  step.ID,
		Success: false,
	}

	log.Printf("⚙️  Executing step %d: %s (tools: %v)", step.ID, step.Description, step.Tools)

	var internetData []InternetSearchResult
	if se.containsTool(step.Tools, "web_search") {
		searchResults, err := se.searcher.Search(ctx, step.Description)
		if err != nil {
			log.Printf("⚠️  Web search failed for step %d: %v", step.ID, err)
		} else {
			internetData = searchResults
			log.Printf("   Found %d search results", len(searchResults))
		}
	}

	prompt := se.buildExecutionPrompt(step, internetData)

	response, err := llms.GenerateFromSinglePrompt(ctx, se.llm, prompt)
	if err != nil {
		result.Error = fmt.Sprintf("LLM execution failed: %v", err)
		log.Printf("❌ Step %d failed: %v", step.ID, err)
		return result, err
	}

	result.Success = true
	result.Result = response
	result.InternetData = internetData

	log.Printf("✅ Step %d complete: %d chars result", step.ID, len(response))

	return result, nil
}

func (se *LLMStepExecutor) containsTool(tools []string, toolName string) bool {
	for _, t := range tools {
		if t == toolName {
			return true
		}
	}
	return false
}

func (se *LLMStepExecutor) buildExecutionPrompt(step ExecutionStep, internetData []InternetSearchResult) string {
	var builder strings.Builder

	builder.WriteString("Execute this task step:\n")
	builder.WriteString(fmt.Sprintf("Task: %s\n", step.Description))

	if len(internetData) > 0 {
		builder.WriteString("\nLatest Information from Internet Search:\n")
		for i, result := range internetData {
			builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
			builder.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
			builder.WriteString(fmt.Sprintf("   Summary: %s\n", result.Snippet))
			if result.Timestamp > 0 {
				t := time.Unix(result.Timestamp, 0)
				builder.WriteString(fmt.Sprintf("   Date: %s\n", t.Format("2006-01-02")))
			}
		}
	}

	builder.WriteString("\nProvide a detailed analysis and response based on:\n")
	builder.WriteString("1. The task requirement\n")
	builder.WriteString("2. Latest internet data (prioritize recent information)\n")
	builder.WriteString("3. Available tools and their capabilities\n")
	builder.WriteString("\nFormat your response as JSON with 'analysis' and 'conclusion' fields.\n")

	return builder.String()
}

type SimpleInternetSearcher struct {
	cache map[string]SearchCacheEntry
	ttl   time.Duration
}

type SearchCacheEntry struct {
	Results   []InternetSearchResult
	Timestamp time.Time
}

func NewSimpleInternetSearcher() *SimpleInternetSearcher {
	return &SimpleInternetSearcher{
		cache: make(map[string]SearchCacheEntry),
		ttl:   10 * time.Minute,
	}
}

func (sis *SimpleInternetSearcher) Search(ctx context.Context, query string) ([]InternetSearchResult, error) {
	if entry, ok := sis.cache[query]; ok {
		if time.Since(entry.Timestamp) < sis.ttl {
			log.Printf("💾 Returning cached search results for: %s", query)
			return entry.Results, nil
		}
	}

	log.Printf("🌐 Searching internet for: %s", query)

	results := []InternetSearchResult{
		{
			Title:      fmt.Sprintf("Search result for: %s", query),
			URL:        fmt.Sprintf("https://search.example.com/q=%s", query),
			Snippet:    "This is a placeholder search result. In production, this would be real data from search engines.",
			Timestamp:  time.Now().Unix(),
			Confidence: 0.8,
		},
	}

	sis.cache[query] = SearchCacheEntry{
		Results:   results,
		Timestamp: time.Now(),
	}

	return results, nil
}

func (sis *SimpleInternetSearcher) ClearCache() {
	log.Printf("🧹 Clearing expired cache entries")

	for key, entry := range sis.cache {
		if time.Since(entry.Timestamp) > sis.ttl {
			delete(sis.cache, key)
		}
	}
}
