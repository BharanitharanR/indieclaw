# Personas Configuration

This directory contains TOML configuration files that define different AI assistant personas. Each persona file specifies:

- **Prompt Template**: The system prompt that defines the AI's behavior and personality
- **LLM Models**: Which models to use for text and vision tasks
- **Tone & Style**: Guidelines for how the AI should communicate
- **WhatsApp Access Control**: Which phone numbers are allowed to interact via WhatsApp

## Quick Start

### Using a Persona

Set the `PERSONA_NAME` environment variable before starting the orchestrator:

```bash
export PERSONA_NAME=executive_coach
./gateway-service/cmd/orchestrator/main
```

If not specified, it defaults to `default.toml`.

## File Format

Each persona TOML file has this structure:

```toml
[persona]
name = "Persona Name"
version = "1.0"

[models]
text_model = "qwen3:8b"
vision_model = "gemma4:e2b"

[personality]
prompt_template = """Your system prompt here..."""
tone = "professional"
max_response_length = 1000
response_style = "detailed"
include_followup_questions = false

[personality.tone_guidelines]
clarity = "Description of tone guideline"
accuracy = "Another guideline"

[whatsapp]
allowed_phone_numbers = [
    "91-98765-43210",
    "91-87654-32109",
]
enabled = true
```

## Creating a New Persona

1. **Copy an existing persona** as a template:
   ```bash
   cp default.toml my_new_persona.toml
   ```

2. **Edit the configuration** to customize:
   - Persona name and description
   - System prompt template
   - Tone and communication style
   - Allowed WhatsApp phone numbers

3. **Test it**:
   ```bash
   export PERSONA_NAME=my_new_persona
   # Start orchestrator and test via WhatsApp
   ```

## Prompt Template Guidelines

Your `prompt_template` can include placeholders that will be automatically filled:

- `%s` (first): Replaced with the list of available tools
- `%s` (second): Replaced with tool names for the Action field

Example:
```toml
prompt_template = """You are a helpful assistant.

Available tools:
%s

When using tools, respond with:
Thought: [reasoning]
Action: [tool name from: %s]
Action Input: [parameters]
Observation: [result]
Final Answer: [your response]
"""
```

## Model Selection

Supported models depend on what's available in your Ollama instance:

- **Text Models**: `qwen3:8b`, `neural-chat:latest`, `mistral:latest`, etc.
- **Vision Models**: `gemma4:e2b`, `llava:latest`, etc.

Check available models: `ollama list`

## WhatsApp Phone Numbers

Format phone numbers in international format, e.g.:
- India: `91-98765-43210`
- US: `1-202-555-0173`
- UK: `44-20-1234-5678`

Numbers can use separators (`-`, spaces, parentheses) - they're normalized during validation.

## Examples

### default.toml
Standard general-purpose assistant. Safe for most use cases.

### executive_coach.toml
Specialized persona for an executive coach providing strategic guidance to clients.
Includes professional tone, actionable insights, and followup questions.

## Environment Variables

- `PERSONA_NAME`: Which persona TOML file to load (default: "default")
- `TEXT_MODEL`: Override persona's text model (optional)
- `VISION_MODEL`: Override persona's vision model (optional)

## Troubleshooting

**Persona file not found:**
```
persona file not found for "my_persona" (tried: [...])
```
Ensure the TOML file exists and is in `gateway-service/config/personas/`

**Invalid TOML syntax:**
```
failed to parse persona TOML: ...
```
Check your TOML file syntax. Use https://www.toml-lint.com/ to validate.

**Phone number not authorized:**
```
❌ Phone number is NOT in the allowed list
```
Add the number to the persona's `[whatsapp] allowed_phone_numbers` list.

## Future Enhancements

Planned features:
- CLI tool to manage personas: `indieclaw persona create`, `edit`, `list`
- Web UI dashboard for persona management
- Hot-reload personas without restarting
- Per-session persona switching
