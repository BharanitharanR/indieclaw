package orchestrator

// PersonaDefinition represents a complete persona configuration
type PersonaDefinition struct {
	Name            string                 `toml:"name"`
	Version         int                    `toml:"version"`
	Description     string                 `toml:"description"`
	PlannerPrompt   string                 `toml:"planner_prompt"`
	Capabilities    PersonaCapabilities    `toml:"capabilities"`
	Intents         map[string]PersonaRule `toml:"intents"`
	ResponseRules   map[string]ResponseRule `toml:"response_rules"`
	ValidationGates []ValidationGate        `toml:"validation_gates"`
	Templates       map[string]string      `toml:"templates"`
	TextModel       string                 `toml:"text_model"`
	VisionModel     string                 `toml:"vision_model"`
}

// PersonaCapabilities defines what this persona can and cannot handle
type PersonaCapabilities struct {
	CanHandle   []string `toml:"can_handle"`
	CannotHandle []string `toml:"cannot_handle"`
	RequiresSearch []string `toml:"requires_search"`
	InternalOnly []string `toml:"internal_only"`
}

// PersonaRule defines behavior for a specific intent
type PersonaRule struct {
	Mode           string  `toml:"mode"`           // "inquiry", "exploration", "decision", "redirect", "escalate"
	Depth          string  `toml:"depth"`          // "deep", "medium", "light"
	SearchEnabled  bool    `toml:"search_enabled"`
	ConfidenceMin  float32 `toml:"confidence_min"`
	ProbeQuestions int     `toml:"probe_questions"`
	Template       string  `toml:"template"`       // Template key for this intent
}

// ResponseRule defines how responses should be formatted for a mode
type ResponseRule struct {
	Mode               string   `toml:"mode"`
	QuestionsRatio     float32  `toml:"questions_ratio"`
	ReflectionRatio    float32  `toml:"reflection_ratio"`
	AdviceRatio        float32  `toml:"advice_ratio"`
	SilenceRatio       float32  `toml:"silence_ratio"`
	MaxLength          int      `toml:"max_length"`
	Tone               string   `toml:"tone"`
	ForbiddenPatterns  []string `toml:"forbidden_patterns"`
	RequiredElements   []string `toml:"required_elements"`
}

// ValidationGate is a rule that responses must pass
type ValidationGate struct {
	Name        string      `toml:"name"`
	GateType    string      `toml:"gate_type"` // "has_element", "ratio_check", "length_check", "confidence_check", "pattern_check"
	Parameters  map[string]interface{} `toml:"parameters"`
	ErrorMessage string      `toml:"error_message"`
	IsCritical  bool        `toml:"is_critical"` // If true, fail on violation; if false, warn
}

// PersonaContext holds the current persona and related state during execution
type PersonaContext struct {
	Definition    *PersonaDefinition
	CurrentIntent string
	CurrentMode   string
	Confidence    float32
	Evidence      map[string]interface{}
}

// PersonaProvider interface for loading personas
type PersonaProvider interface {
	LoadPersona(name string) (*PersonaDefinition, error)
	ListPersonas() []string
	GetCurrentPersona() *PersonaDefinition
	ValidatePersonaConfig(p *PersonaDefinition) error
}

// PersonaExecutor interface for executing persona logic
type PersonaExecutor interface {
	ClassifyIntent(message string, context *PersonaContext) (string, float32, error)
	GetModeForIntent(intent string, persona *PersonaDefinition) PersonaRule
	ValidateResponseAgainstRules(response string, mode string, persona *PersonaDefinition) []ValidationIssue
	ApplyResponseTemplate(template string, data map[string]interface{}) string
}

// ValidationIssue represents a problem found during response validation
type ValidationIssue struct {
	Gate     string
	Severity string // "error", "warning"
	Message  string
	Fix      string // Suggested fix
}

// PersonaDecision represents the result of persona decision logic
type PersonaDecision struct {
	ShouldProceed      bool
	Intent             string
	Mode               string
	Confidence         float32
	RecommendedAction  string
	FallbackPersona    string // If should escalate to different persona
	ValidationRules    []ValidationGate
	ResponseTemplate   string
}
