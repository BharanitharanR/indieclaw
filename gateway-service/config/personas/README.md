# Indieclaw Persona Configuration

This directory contains persona definitions for Indieclaw. Each persona is a TOML file that defines how the assistant should behave for a specific use case.

## Quick Start

To add a new persona:
1. Copy `generic_assistant.toml` as a template
2. Edit the persona name, description, and intents
3. Add it to the personas directory
4. Load it with `PERSONA_NAME=your_persona_name`

## TOML Schema

### Top-Level Fields

```toml
name = "Executive Coach"              # Display name
version = 1                            # Semantic version (starts at 1)
description = "..."                    # What this persona does
text_model = "qwen2:7b"               # LLM for text generation
vision_model = "llava:7b"             # LLM for image understanding
```

### Capabilities Section

Defines what the persona can and cannot handle:

```toml
[capabilities]
can_handle = [
    "career_decisions",
    "leadership_challenges",
    # ... topics this persona handles
]

cannot_handle = [
    "mental_health_crisis",
    # ... topics that should be escalated
]

requires_search = [
    "market_data",
    # ... topics that need web search
]

internal_only = [
    "reflection_questions",
    # ... topics that need only reasoning
]
```

### Intents Section

Each intent is a classification category for user messages. Define one intent per section:

```toml
[intents.career_decision]
mode = "inquiry"                    # Response mode: inquiry, exploration, reframing, redirect, etc.
depth = "deep"                      # Coaching depth: deep, medium, light
search_enabled = false              # Use web search for this intent?
confidence_min = 0.6                # Minimum confidence to proceed (0-1)
probe_questions = 3                 # Number of follow-up questions to ask
template = "career_inquiry"         # Response template key to use
```

**Available Modes:**
- `inquiry` - Ask questions to understand the situation
- `exploration` - Explore different perspectives
- `reframing` - Help see things from a new angle
- `decision` - Guide decision-making
- `research` - Provide information/research
- `redirect` - Redirect to coaching
- `referral` - Escalate to appropriate service
- `education` - Teach/explain
- `clarify` - Ask for clarification

### Response Rules Section

Define how responses should be formatted for each mode:

```toml
[response_rules.inquiry]
mode = "inquiry"
questions_ratio = 0.60              # 0-1: proportion of response that should be questions
reflection_ratio = 0.30             # 0-1: proportion that should be reflections
advice_ratio = 0.10                 # 0-1: proportion that should be advice
silence_ratio = 0.0                 # 0-1: proportion of silence (for thinking)
max_length = 300                    # Maximum characters
tone = "curious"                    # Tone: curious, warm, professional, insightful, etc.
forbidden_patterns = [              # Phrases to avoid
    "you should",
    "I recommend"
]
required_elements = [               # Elements that must be present
    "question"
]
```

**Important:** All ratio fields must sum to 1.0 (allowing small floating-point error).

### Validation Gates Section

Define rules that responses must pass:

```toml
[[validation_gates]]
name = "max_questions_limit"
gate_type = "length_check"          # Type: has_element, ratio_check, length_check, pattern_check
parameters = { max_length = 350 }
error_message = "Response too long"
is_critical = true                  # If true, fail on violation; if false, warn only
```

**Gate Types:**
- `has_element` - Check if response contains required element
- `ratio_check` - Validate response follows defined ratios
- `length_check` - Validate response length
- `pattern_check` - Check for forbidden patterns
- `confidence_check` - Validate confidence threshold

### Templates Section

Define response templates that can be used:

```toml
[templates]
career_inquiry = "Tell me more about what's drawing you toward this direction. What would success look like to you?"
```

Templates can include placeholders: `{{variable_name}}`

## Complete Example

See `executive_coach.toml` for a complete production example.

## Loading a Persona

Set the `PERSONA_NAME` environment variable:

```bash
export PERSONA_NAME=executive_coach
./start-orchestrator.sh
```

Or pass it inline:

```bash
PERSONA_NAME=executive_coach go run ./cmd/orchestrator/main.go
```

## Multiple Personas

You can have multiple persona files. Indieclaw supports:
- **Runtime selection** via `PERSONA_NAME` env var
- **API endpoint** `/personas` to list available personas
- **API endpoint** `/personas/{name}` to get persona details
- **Switching** without restarting (coming soon)

## Best Practices

### 1. Start with a Copy
Always copy an existing persona as a template rather than starting from scratch.

### 2. Define Intents Clearly
Each intent should map to a specific user behavior or question type. Avoid overlapping intents.

### 3. Be Strict with Confidence
Set `confidence_min` high (0.7+) if this is a specialized persona. Lower confidence (0.4-0.5) for general personas.

### 4. Use Validation Gates
Define what makes a "good" response for your persona. Don't skip this.

### 5. Test the Persona
Before deploying:
```bash
# Test by sending messages via WhatsApp
# Check logs to see intent classification and confidence
# Verify responses match expected tone and structure
```

## Troubleshooting

### Persona Not Loading
```
Error: failed to load persona "my_coach": no such file or directory
```
→ Check the filename is lowercase and ends with `.toml`

### Validation Errors
```
Error: invalid persona configuration: intents missing required field
```
→ Check TOML syntax (use a TOML validator)
→ Verify all required fields are present

### Intent Misclassification
If the persona is routing to wrong intents:
- Check confidence scores in logs
- Verify intent descriptions are distinct
- Add more context to intent templates

## Schema Validation

To validate a persona TOML file:

```bash
cd gateway-service
go run ./cmd/orchestrator/main.go --validate-persona config/personas/my_persona.toml
```

(This feature coming soon)

## Adding Custom Intent Types

To add new intent types (e.g., "negotiation", "conflict_resolution"):

1. Add to the persona TOML:
```toml
[intents.negotiation]
mode = "collaboration"
# ...
```

2. Ensure the mode is defined in response_rules
3. Optionally add validation gates for that mode
4. Test with real examples

## Extending the System

The persona configuration system is designed to be extended. Future enhancements:
- **Conditional routing** - Route based on conversation context
- **Multi-layer composition** - Stack personas (e.g., "coach + empathy layer")
- **A/B testing** - Swap persona versions without code changes
- **Custom rule types** - Define domain-specific rules
- **Hot reloading** - Swap personas without restart

For now, personas are static after startup but can be versioned (e.g., `executive_coach_v1`, `executive_coach_v2`).
