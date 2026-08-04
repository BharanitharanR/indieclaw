package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/toml"
)

var (
	personaDefinitionCache = make(map[string]*PersonaDefinition)
	personaCacheMutex      sync.RWMutex
	currentPersonaDefinition *PersonaDefinition
)

// LoadPersonaDefinition loads a new-style persona from TOML file by name
// This is the new generic persona system with intent-based routing
func LoadPersonaDefinition(name string) (*PersonaDefinition, error) {
	// Check cache first
	personaCacheMutex.RLock()
	if cached, ok := personaDefinitionCache[name]; ok {
		personaCacheMutex.RUnlock()
		return cached, nil
	}
	personaCacheMutex.RUnlock()

	// Construct path to persona file
	configDir := os.Getenv("PERSONA_CONFIG_DIR")
	if configDir == "" {
		// Default to gateway-service/config/personas
		configDir = filepath.Join("gateway-service", "config", "personas")
	}

	personaFile := filepath.Join(configDir, name+".toml")

	// Read and parse TOML
	var persona PersonaDefinition
	if _, err := toml.DecodeFile(personaFile, &persona); err != nil {
		return nil, fmt.Errorf("failed to load persona %q from %s: %w", name, personaFile, err)
	}

	// Validate configuration
	if err := ValidatePersonaConfig(&persona); err != nil {
		return nil, fmt.Errorf("invalid persona configuration for %q: %w", name, err)
	}

	// Cache it
	personaCacheMutex.Lock()
	personaDefinitionCache[name] = &persona
	personaCacheMutex.Unlock()

	return &persona, nil
}

// SetCurrentPersonaDefinition sets the active persona for this process
func SetCurrentPersonaDefinition(name string) error {
	persona, err := LoadPersonaDefinition(name)
	if err != nil {
		// Fallback to generic persona
		genericPersona, err2 := LoadPersonaDefinition("generic_assistant")
		if err2 != nil {
			return fmt.Errorf("failed to load persona %q and fallback generic_assistant: %w, %w", name, err, err2)
		}
		currentPersonaDefinition = genericPersona
		return nil
	}

	currentPersonaDefinition = persona
	return nil
}

// GetCurrentPersonaDefinition returns the currently active persona definition
func GetCurrentPersonaDefinition() *PersonaDefinition {
	if currentPersonaDefinition == nil {
		// Initialize with generic persona
		if err := SetCurrentPersonaDefinition("generic_assistant"); err != nil {
			panic(fmt.Sprintf("failed to load generic persona: %v", err))
		}
	}
	return currentPersonaDefinition
}

// LoadPersona is a wrapper that tries to load PersonaDefinition first,
// then falls back to PersonaConfig for backward compatibility
func LoadPersona(name string) (*PersonaDefinition, error) {
	return LoadPersonaDefinition(name)
}

// ListAvailablePersonas lists all persona files in the config directory
func ListAvailablePersonas() ([]string, error) {
	configDir := os.Getenv("PERSONA_CONFIG_DIR")
	if configDir == "" {
		configDir = filepath.Join("gateway-service", "config", "personas")
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read personas directory %s: %w", configDir, err)
	}

	var personas []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".toml" {
			name := entry.Name()[:len(entry.Name())-5] // Remove .toml extension
			personas = append(personas, name)
		}
	}

	return personas, nil
}

// ValidatePersonaConfig checks persona configuration for semantic errors
func ValidatePersonaConfig(p *PersonaDefinition) error {
	if p.Name == "" {
		return fmt.Errorf("persona must have a name")
	}

	if p.Version < 1 {
		return fmt.Errorf("persona version must be >= 1")
	}

	// Validate intents
	if len(p.Intents) == 0 {
		return fmt.Errorf("persona must define at least one intent")
	}

	for intentName, rule := range p.Intents {
		if rule.Mode == "" {
			return fmt.Errorf("intent %q missing mode", intentName)
		}

		// Validate mode is known
		validModes := []string{"inquiry", "exploration", "decision", "redirect", "escalate", "research"}
		if !contains(validModes, rule.Mode) {
			return fmt.Errorf("intent %q has invalid mode %q", intentName, rule.Mode)
		}

		// Validate confidence threshold
		if rule.ConfidenceMin < 0 || rule.ConfidenceMin > 1 {
			return fmt.Errorf("intent %q confidence_min must be between 0 and 1", intentName)
		}
	}

	// Validate response rules
	for modeName, rule := range p.ResponseRules {
		total := rule.QuestionsRatio + rule.ReflectionRatio + rule.AdviceRatio + rule.SilenceRatio
		if total < 0.99 || total > 1.01 { // Allow small floating point error
			return fmt.Errorf("response rule %q ratios must sum to 1.0, got %f", modeName, total)
		}

		if rule.MaxLength <= 0 {
			return fmt.Errorf("response rule %q max_length must be > 0", modeName)
		}
	}

	// Validate validation gates
	for i, gate := range p.ValidationGates {
		if gate.Name == "" {
			return fmt.Errorf("validation gate %d missing name", i)
		}

		validGateTypes := []string{"has_element", "ratio_check", "length_check", "confidence_check", "pattern_check"}
		if !contains(validGateTypes, gate.GateType) {
			return fmt.Errorf("validation gate %q has invalid type %q", gate.Name, gate.GateType)
		}
	}

	// Validate templates exist for referenced intents
	for intentName, rule := range p.Intents {
		if rule.Template != "" {
			if _, ok := p.Templates[rule.Template]; !ok {
				return fmt.Errorf("intent %q references template %q which does not exist", intentName, rule.Template)
			}
		}
	}

	return nil
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ClearPersonaCache clears the in-memory persona cache (useful for testing)
func ClearPersonaCache() {
	personaCacheMutex.Lock()
	defer personaCacheMutex.Unlock()
	personaCache = make(map[string]*PersonaDefinition)
	currentPersona = nil
}

// GetPersonaStats returns statistics about cached personas
func GetPersonaStats() map[string]interface{} {
	personaCacheMutex.RLock()
	defer personaCacheMutex.RUnlock()

	var currentName string
	if currentPersonaDefinition != nil {
		currentName = currentPersonaDefinition.Name
	}

	return map[string]interface{}{
		"cached_count":      len(personaDefinitionCache),
		"current_persona":   currentName,
		"cached_names":      getPersonaNames(),
	}
}

func getPersonaNames() []string {
	var names []string
	for name := range personaDefinitionCache {
		names = append(names, name)
	}
	return names
}
