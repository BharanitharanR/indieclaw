package orchestrator

import (
	"context"
	"fmt"
	"log"

	"github.com/qdrant/go-client/qdrant"
	"github.com/tmc/langchaingo/llms"
)

type Pipeline struct {
	contextRetriever *QdrantContextRetriever
	tokenManager     *SimpleTokenManager
	planner          *LLMPlanner
	confidenceGate   *ConfidenceValidator
	stepRefinement   *StepRefinement
	stepExecutor     *LLMStepExecutor
	resultCoalator   *LLMResultCoalator
	responseFormatter *ResponseFormatter
}

func NewPipeline(
	qdrantClient *qdrant.Client,
	plannerLLM llms.Model,
	mainLLM llms.Model,
	embeddingModel string,
) *Pipeline {
	searcher := NewSimpleInternetSearcher()

	return &Pipeline{
		contextRetriever: NewQdrantContextRetriever(qdrantClient, embeddingModel),
		tokenManager:     NewSimpleTokenManager(),
		planner:          NewLLMPlanner(plannerLLM),
		confidenceGate:   NewConfidenceValidator(50.0),
		stepRefinement:   NewStepRefinement(mainLLM),
		stepExecutor:     NewLLMStepExecutor(mainLLM, searcher),
		resultCoalator:   NewLLMResultCoalator(mainLLM),
		responseFormatter: NewResponseFormatter("text"),
	}
}

func (p *Pipeline) Execute(ctx context.Context, input string, sessionID string, persona *PersonaConfig) (string, error) {
	pctx := &PipelineContext{
		SessionID: sessionID,
		UserInput: input,
		Persona:   persona,
	}

	log.Printf("\n=== PIPELINE START ===")
	log.Printf("📥 Input: %s", input)
	log.Printf("👤 Persona: %s", persona.Name)
	log.Printf("🔑 Session: %s\n", sessionID)

	if err := p.stage1_RetrieveContext(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Context Retrieval", err)
	}

	if err := p.stage2_TokenizeContext(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Tokenization", err)
	}

	if err := p.stage3_Plan(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Planning", err)
	}

	if err := p.stage4_ConfidenceGate(ctx, pctx); err != nil {
		return "", nil
	}

	if err := p.stage5_RefineSteps(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Step Refinement", err)
	}

	if err := p.stage6_ExecuteSteps(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Step Execution", err)
	}

	if err := p.stage7_CoalesceResults(ctx, pctx); err != nil {
		return "", p.handleError(pctx, "Result Coalation", err)
	}

	log.Printf("\n=== PIPELINE COMPLETE ===\n")

	return pctx.FinalResponse, nil
}

func (p *Pipeline) stage1_RetrieveContext(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[1/7] 🔍 CONTEXT RETRIEVAL")

	contexts, err := p.contextRetriever.Retrieve(ctx, pctx.UserInput, pctx.SessionID, 5)
	if err != nil {
		log.Printf("⚠️  Context retrieval failed: %v (continuing without context)", err)
		contexts = []RetrievedContext{}
	}

	pctx.RetrievedContexts = contexts
	log.Printf("    ✅ Retrieved %d contexts\n", len(contexts))

	return nil
}

func (p *Pipeline) stage2_TokenizeContext(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[2/7] 📊 TOKENIZATION & TRIMMING")

	compressor := NewContextCompressor()
	trimmedCtx, err := compressor.CompressContexts(pctx.RetrievedContexts, 2000, 0.7)
	if err != nil {
		return fmt.Errorf("context compression failed: %w", err)
	}

	pctx.TrimmedContext = trimmedCtx

	totalTokens, _ := p.tokenManager.EstimateTotalTokens(pctx.UserInput, pctx.TrimmedContext, pctx.Persona)
	pctx.TokenCount = totalTokens

	log.Printf("    ✅ Total tokens: %d\n", totalTokens)

	return nil
}

func (p *Pipeline) stage3_Plan(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[3/7] 🤔 PLANNER - FEASIBILITY CHECK")

	planResp, err := p.planner.Plan(ctx, pctx.UserInput, pctx.TrimmedContext, pctx.Persona)
	if err != nil {
		return fmt.Errorf("planner failed: %w", err)
	}

	pctx.PlannerResponse = planResp
	pctx.Confidence = planResp.Confidence

	if planResp.PiiWarning != "" {
		log.Printf("    ⚠️  PII Warning: %s", planResp.PiiWarning)
	}

	log.Printf("    ✅ Feasibility: %.0f%% | Can help: %v\n", planResp.Confidence, planResp.CanHelp)

	return nil
}

func (p *Pipeline) stage4_ConfidenceGate(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[4/7] 🚪 CONFIDENCE GATE (threshold: 50%%)")

	hasPII := pctx.PlannerResponse.PiiWarning != ""
	shouldProceed := p.confidenceGate.ShouldProceed(pctx.Confidence, hasPII)

	if !shouldProceed {
		log.Printf("    ❌ REJECTED: Confidence %.0f%% below threshold\n", pctx.Confidence)
		pctx.FinalResponse = p.confidenceGate.RejectionMessage()
		return fmt.Errorf("confidence too low")
	}

	log.Printf("    ✅ APPROVED: Proceeding\n")
	return nil
}

func (p *Pipeline) stage5_RefineSteps(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[5/7] 🔧 STEP DECOMPOSITION")

	steps, err := p.stepRefinement.RefineSteps(ctx, pctx.PlannerResponse.Steps, pctx.UserInput)
	if err != nil {
		return fmt.Errorf("step decomposition failed: %w", err)
	}

	pctx.ExecutionPlan = steps
	log.Printf("    ✅ Created %d execution steps\n", len(steps))

	return nil
}

func (p *Pipeline) stage6_ExecuteSteps(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[6/7] ⚙️  STEP EXECUTION (with internet search)")

	for _, step := range pctx.ExecutionPlan {
		stepResult, err := p.stepExecutor.ExecuteStep(ctx, step, nil)
		if err != nil {
			log.Printf("    ⚠️  Step %d execution error: %v", step.ID, err)
		}

		pctx.StepResults = append(pctx.StepResults, stepResult)
	}

	log.Printf("    ✅ Executed %d steps\n", len(pctx.StepResults))

	return nil
}

func (p *Pipeline) stage7_CoalesceResults(ctx context.Context, pctx *PipelineContext) error {
	log.Printf("[7/7] 🔗 RESULT COALATION")

	finalResp, err := p.resultCoalator.Coalesce(ctx, pctx.StepResults, pctx.Persona)
	if err != nil {
		return fmt.Errorf("result coalation failed: %w", err)
	}

	formatted := p.responseFormatter.Format(finalResp)

	pctx.FinalResponse = formatted
	log.Printf("    ✅ Final response: %d chars\n", len(formatted))

	return nil
}

func (p *Pipeline) handleError(pctx *PipelineContext, stage string, err error) error {
	msg := fmt.Sprintf("Error in %s: %v", stage, err)
	log.Printf("❌ %s\n", msg)

	pctx.Errors = append(pctx.Errors, msg)

	return fmt.Errorf("%s", msg)
}

func (p *Pipeline) GetPipelineMetrics(pctx *PipelineContext) map[string]interface{} {
	return map[string]interface{}{
		"session_id":           pctx.SessionID,
		"input_length":         len(pctx.UserInput),
		"token_count":          pctx.TokenCount,
		"confidence":           pctx.Confidence,
		"retrieved_contexts":   len(pctx.RetrievedContexts),
		"execution_steps":      len(pctx.ExecutionPlan),
		"completed_steps":      len(pctx.StepResults),
		"final_response_length": len(pctx.FinalResponse),
		"errors":               len(pctx.Errors),
	}
}
