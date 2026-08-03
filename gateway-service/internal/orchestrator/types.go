package orchestrator

import "context"

type PipelineContext struct {
	SessionID          string
	UserInput          string
	IsVision           bool
	ImageBase64        string
	Persona            *PersonaConfig
	RetrievedContexts  []RetrievedContext
	TrimmedContext     string
	TokenCount         int
	PlannerResponse    PlannerResponse
	Confidence         float64
	ExecutionPlan      []ExecutionStep
	StepResults        []StepResult
	FinalResponse      string
	Errors             []string
}

type RetrievedContext struct {
	Content    string
	Similarity float64
	Source     string
	SessionID  string
}

type PlannerResponse struct {
	CanHelp   bool
	Confidence float64
	Reasoning string
	Steps     []string
	PiiWarning string
}

type ExecutionStep struct {
	ID          int
	Description string
	Action      string
	Tools       []string
	Input       map[string]interface{}
	Result      string
	InternetData []InternetSearchResult
}

type StepResult struct {
	StepID       int
	Success      bool
	Result       string
	InternetData []InternetSearchResult
	Error        string
}

type InternetSearchResult struct {
	Title       string
	URL         string
	Snippet     string
	Timestamp   int64
	Confidence  float64
}

type PipelineStage interface {
	Execute(ctx context.Context, pctx *PipelineContext) error
}

type ContextRetriever interface {
	Retrieve(ctx context.Context, input string, sessionID string, limit int) ([]RetrievedContext, error)
}

type TokenManager interface {
	CountTokens(text string) (int, error)
	TrimContext(context string, maxTokens int) (string, error)
	EstimateTotalTokens(input string, context string, persona *PersonaConfig) (int, error)
}

type PlannerInterface interface {
	Plan(ctx context.Context, input string, context string, persona *PersonaConfig) (PlannerResponse, error)
}

type ConfidenceGate interface {
	ShouldProceed(confidence float64, pii bool) bool
	RejectionMessage() string
}

type StepDecomposer interface {
	Decompose(plannerSteps []string, input string, context string) ([]ExecutionStep, error)
}

type StepExecutor interface {
	ExecuteStep(ctx context.Context, step ExecutionStep, tools map[string]Tool) (StepResult, error)
}

type InternetSearcher interface {
	Search(ctx context.Context, query string) ([]InternetSearchResult, error)
}

type ResultCoalator interface {
	Coalesce(ctx context.Context, steps []StepResult, persona *PersonaConfig) (string, error)
}

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}
