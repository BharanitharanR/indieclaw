package orchestrator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

type PersonaConfig struct {
	Name                string            `toml:"name"`
	Version             string            `toml:"version"`
	TextModel           string            `toml:"text_model"`
	VisionModel         string            `toml:"vision_model"`
	PromptTemplate      string            `toml:"prompt_template"`
	PlannerPrompt       string            `toml:"planner_prompt"`
	Tone                string            `toml:"tone"`
	MaxResponseLength   int               `toml:"max_response_length"`
	ResponseStyle       string            `toml:"response_style"`
	IncludeFollowup     bool              `toml:"include_followup_questions"`
	ToneGuidelines      map[string]string `toml:"tone_guidelines"`
	AllowedPhoneNumbers []string          `toml:"allowed_phone_numbers"`
	Enabled             bool              `toml:"enabled"`
}

type PersonaFile struct {
	Persona    PersonaConfig `toml:"persona"`
	Models     ModelConfig   `toml:"models"`
	Personality PersonalityConfig `toml:"personality"`
	WhatsApp   WhatsAppConfig `toml:"whatsapp"`
}

type ModelConfig struct {
	TextModel   string `toml:"text_model"`
	VisionModel string `toml:"vision_model"`
}

type PersonalityConfig struct {
	PromptTemplate      string            `toml:"prompt_template"`
	PlannerPrompt       string            `toml:"planner_prompt"`
	Tone                string            `toml:"tone"`
	MaxResponseLength   int               `toml:"max_response_length"`
	ResponseStyle       string            `toml:"response_style"`
	IncludeFollowup     bool              `toml:"include_followup_questions"`
	ToneGuidelines      map[string]string `toml:"tone_guidelines"`
}

type WhatsAppConfig struct {
	AllowedPhoneNumbers []string `toml:"allowed_phone_numbers"`
	Enabled             bool     `toml:"enabled"`
}

var (
	currentPersona *PersonaConfig
	personaMutex   sync.RWMutex
)

// LoadPersona loads a persona from a TOML file in the config/personas directory
// personaName should be the filename without .toml extension (e.g., "default" or "executive_coach")
func LoadPersona(personaName string) (*PersonaConfig, error) {
	if personaName == "" {
		personaName = "default"
	}

	configPath := fmt.Sprintf("config/personas/%s.toml", personaName)

	// Try multiple possible config directories
	possiblePaths := []string{
		configPath,
		filepath.Join("gateway-service", configPath),
		filepath.Join("..", "..", configPath),
	}

	var data []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		data, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if data == nil {
		return nil, fmt.Errorf("persona file not found for %q (tried: %v)", personaName, possiblePaths)
	}

	log.Printf("📂 Loading persona from: %s", foundPath)

	var personaFile PersonaFile
	err = toml.Unmarshal(data, &personaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse persona TOML: %w", err)
	}

	// Merge the configuration from different sections
	persona := &PersonaConfig{
		Name:                personaFile.Persona.Name,
		Version:             personaFile.Persona.Version,
		TextModel:           personaFile.Models.TextModel,
		VisionModel:         personaFile.Models.VisionModel,
		PromptTemplate:      personaFile.Personality.PromptTemplate,
		PlannerPrompt:       personaFile.Personality.PlannerPrompt,
		Tone:                personaFile.Personality.Tone,
		MaxResponseLength:   personaFile.Personality.MaxResponseLength,
		ResponseStyle:       personaFile.Personality.ResponseStyle,
		IncludeFollowup:     personaFile.Personality.IncludeFollowup,
		ToneGuidelines:      personaFile.Personality.ToneGuidelines,
		AllowedPhoneNumbers: personaFile.WhatsApp.AllowedPhoneNumbers,
		Enabled:             personaFile.WhatsApp.Enabled,
	}

	if persona.Name == "" {
		persona.Name = personaName
	}

	if persona.TextModel == "" {
		persona.TextModel = "qwen3:8b"
	}

	if persona.VisionModel == "" {
		persona.VisionModel = "gemma4:e2b"
	}

	if persona.PromptTemplate == "" {
		return nil, fmt.Errorf("persona %q has empty prompt_template", personaName)
	}

	// Cache the loaded persona
	personaMutex.Lock()
	currentPersona = persona
	personaMutex.Unlock()

	log.Printf("✅ Persona loaded: %s | Models: %s (text), %s (vision) | Allowed numbers: %d",
		persona.Name, persona.TextModel, persona.VisionModel, len(persona.AllowedPhoneNumbers))

	return persona, nil
}

// GetPersona returns the currently loaded persona
// Returns nil if no persona is loaded yet
func GetPersona() *PersonaConfig {
	personaMutex.RLock()
	defer personaMutex.RUnlock()
	return currentPersona
}

// ValidatePhoneNumber checks if a phone number is in the allowed list
func ValidatePhoneNumber(phoneNumber string) bool {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil || !currentPersona.Enabled {
		log.Printf("⚠️  No persona loaded or WhatsApp is disabled")
		return false
	}

	if len(currentPersona.AllowedPhoneNumbers) == 0 {
		log.Printf("⚠️  No phone numbers allowed in current persona")
		return false
	}

	// Normalize phone number for comparison
	normalizedInput := normalizePhoneNumber(phoneNumber)

	for _, allowed := range currentPersona.AllowedPhoneNumbers {
		if normalizePhoneNumber(allowed) == normalizedInput {
			log.Printf("✅ Phone number %q is allowed", phoneNumber)
			return true
		}
	}

	log.Printf("❌ Phone number %q is NOT in the allowed list", phoneNumber)
	return false
}

// normalizePhoneNumber removes common separators for comparison
func normalizePhoneNumber(number string) string {
	number = strings.ToLower(number)
	number = strings.ReplaceAll(number, "-", "")
	number = strings.ReplaceAll(number, " ", "")
	number = strings.ReplaceAll(number, "(", "")
	number = strings.ReplaceAll(number, ")", "")
	number = strings.ReplaceAll(number, "+", "")
	return number
}

// GetAllowedPhoneNumbers returns the list of allowed WhatsApp numbers
func GetAllowedPhoneNumbers() []string {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil {
		return []string{}
	}
	return currentPersona.AllowedPhoneNumbers
}

// GetTextModel returns the configured text model
func GetTextModel() string {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil {
		return "qwen3:8b"
	}
	return currentPersona.TextModel
}

// GetVisionModel returns the configured vision model
func GetVisionModel() string {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil {
		return "gemma4:e2b"
	}
	return currentPersona.VisionModel
}

// GetPromptTemplate returns the configured prompt template
func GetPromptTemplate() string {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil {
		return ""
	}
	return currentPersona.PromptTemplate
}

// GetPlannerPrompt returns the configured planner prompt
func GetPlannerPrompt() string {
	personaMutex.RLock()
	defer personaMutex.RUnlock()

	if currentPersona == nil {
		return ""
	}
	if currentPersona.PlannerPrompt != "" {
		return currentPersona.PlannerPrompt
	}
	// Fallback to default if not configured
	return defaultPlannerPrompt()
}

func defaultPlannerPrompt() string {
	return `You are a STRICT feasibility analyzer. Your job is to reject questions outside the persona's expertise.

USER QUESTION: "%s"

PERSONA SCOPE & INSTRUCTIONS:
%s

EVALUATION RULES:
1. If the question has NOTHING to do with the persona's domain, set confidence < 30%
2. Only accept if you're CERTAIN the persona can provide EXPERT guidance
3. Be CRITICAL, not helpful. A general LLM can answer trivia - your job is to gatekeep expertise
4. Extract any PII mentioned

Respond JSON:
{
  "can_help": boolean,
  "confidence": number (0-100, be strict),
  "reasoning": "...",
  "pii_warning": "if any PII detected",
  "steps": []
}`
}
